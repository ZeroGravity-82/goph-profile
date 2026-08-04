#!/usr/bin/env bash
# Скрипт завершается при первой ошибке, обращении к неопределенной переменной или ошибке внутри конвейера.
set -euo pipefail

# Поднимает воспроизводимое локальное окружение GophProfile в Rancher Desktop.

# Абсолютный путь позволяет запускать скрипт из любого текущего каталога.
ROOT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"

# Контексты можно переопределить, не изменяя сам скрипт.
KUBE_CONTEXT="${KUBE_CONTEXT:-rancher-desktop}"
DOCKER_CONTEXT="${DOCKER_CONTEXT:-rancher-desktop}"
HELM_BIN="${HELM_BIN:-helm}"

# Приложение и стек мониторинга устанавливаются как независимые Helm-релизы в разных namespace.
APP_NAMESPACE="goph-profile"
APP_RELEASE="goph-profile"
MONITORING_NAMESPACE="monitoring"
MONITORING_RELEASE="monitoring"

# Фиксированная версия делает локальное окружение воспроизводимым при повторной установке.
PROMETHEUS_STACK_VERSION="88.1.3"
DEPENDENCIES_FILE="${ROOT_DIR}/deploy/kubernetes/local/dependencies.yaml"
MONITORING_VALUES_FILE="${ROOT_DIR}/deploy/kubernetes/local/kube-prometheus-stack.values.yaml"
APP_VALUES_FILE="${ROOT_DIR}/helm/goph-profile/values.local.yaml"

usage() {
  cat <<'EOF'
Использование: scripts/kubernetes-local.sh <up|down|status>

  up      собрать образ и развернуть зависимости, мониторинг и GophProfile
  down    удалить Helm-релизы и остановить локальные зависимости, сохранив PVC
  status  показать состояние локального Kubernetes-окружения

Переменные окружения:
  KUBE_CONTEXT     Kubernetes-контекст (по умолчанию rancher-desktop)
  DOCKER_CONTEXT   Docker-контекст (по умолчанию rancher-desktop)
  HELM_BIN         путь к Helm 3 (по умолчанию helm)
  SKIP_BUILD=1     не пересобирать локальный Docker-образ
EOF
}

log() {
  printf '\n==> %s\n' "$1"
}

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    printf 'Не найдена команда: %s\n' "$1" >&2
    exit 1
  fi
}

require_base_commands() {
  require_command kubectl
  require_command "${HELM_BIN}"
}

require_helm_3() {
  local version
  version="$("${HELM_BIN}" version --short)"
  # Helm 4 в текущем окружении некорректно ожидает завершения миграционного хука.
  if [[ ! "${version}" =~ ^v3\. ]]; then
    printf 'Для воспроизводимого запуска требуется Helm 3, найден %s. Укажите путь через HELM_BIN.\n' "${version}" >&2
    exit 1
  fi
}

check_cluster() {
  kubectl --context "${KUBE_CONTEXT}" cluster-info >/dev/null
}

ensure_namespace() {
  local namespace="$1"
  # Генерация манифеста и последующий apply делают создание namespace идемпотентным.
  kubectl --context "${KUBE_CONTEXT}" create namespace "${namespace}" \
    --dry-run=client \
    --output=yaml | kubectl --context "${KUBE_CONTEXT}" apply --filename=-
}

build_image() {
  if [[ "${SKIP_BUILD:-0}" == "1" ]]; then
    log "Сборка образа пропущена"
    return
  fi

  require_command docker
  docker --context "${DOCKER_CONTEXT}" info >/dev/null

  # Rancher Desktop использует этот Docker-контекст, поэтому образ сразу доступен Kubernetes без реестра образов.
  log "Сборка образа goph-profile:local"
  docker --context "${DOCKER_CONTEXT}" build \
    --file "${ROOT_DIR}/docker/Dockerfile" \
    --tag goph-profile:local \
    "${ROOT_DIR}"
}

ensure_tls_certificates() {
  # Существующая действующая для локального запуска TLS-пара не перезаписывается.
  if [[ -s "${ROOT_DIR}/certs/ca.crt" && -s "${ROOT_DIR}/certs/server.crt" && -s "${ROOT_DIR}/certs/server.key" ]]; then
    return
  fi

  require_command openssl
  log "Генерация локальных TLS-сертификатов"
  "${ROOT_DIR}/certs/generate-local-certs.sh"
}

deploy_dependencies() {
  log "Развертывание внешних зависимостей в namespace ${APP_NAMESPACE}"
  kubectl --context "${KUBE_CONTEXT}" apply --filename "${DEPENDENCIES_FILE}"

  # Приложение и миграции запускаются только после готовности всех необходимых сервисов.
  local deployment
  for deployment in postgresql minio rabbitmq opensearch jaeger; do
    # После команды down Deployment остаются в кластере с replicas=0, чтобы сохранить PVC.
    kubectl --context "${KUBE_CONTEXT}" scale "deployment/${deployment}" \
      --namespace "${APP_NAMESPACE}" \
      --replicas=1
    kubectl --context "${KUBE_CONTEXT}" rollout status \
      "deployment/${deployment}" \
      --namespace "${APP_NAMESPACE}" \
      --timeout=10m
  done
}

deploy_monitoring() {
  # Prometheus Operator должен установить CRD до создания ServiceMonitor и PrometheusRule приложения.
  log "Установка kube-prometheus-stack ${PROMETHEUS_STACK_VERSION}"
  "${HELM_BIN}" repo add prometheus-community https://prometheus-community.github.io/helm-charts \
    --force-update
  "${HELM_BIN}" repo update prometheus-community
  "${HELM_BIN}" --kube-context "${KUBE_CONTEXT}" upgrade --install "${MONITORING_RELEASE}" \
    prometheus-community/kube-prometheus-stack \
    --namespace "${MONITORING_NAMESPACE}" \
    --create-namespace \
    --version "${PROMETHEUS_STACK_VERSION}" \
    --values "${MONITORING_VALUES_FILE}" \
    --wait \
    --timeout 15m
}

deploy_application() {
  ensure_tls_certificates

  # Сертификат и закрытый ключ передаются напрямую Helm и не записываются в values-файл.
  log "Установка Helm-релиза GophProfile"
  "${HELM_BIN}" --kube-context "${KUBE_CONTEXT}" upgrade --install "${APP_RELEASE}" \
    "${ROOT_DIR}/helm/goph-profile" \
    --namespace "${APP_NAMESPACE}" \
    --values "${APP_VALUES_FILE}" \
    --set-file "tls.certificate=${ROOT_DIR}/certs/server.crt" \
    --set-file "tls.privateKey=${ROOT_DIR}/certs/server.key" \
    --wait \
    --timeout 10m
}

show_status() {
  require_base_commands
  check_cluster

  log "Helm-релизы"
  "${HELM_BIN}" --kube-context "${KUBE_CONTEXT}" list --all-namespaces

  log "GophProfile и локальные зависимости"
  # Одной командой выводятся компоненты, сетевые точки входа, хранилища и состояние автомасштабирования.
  kubectl --context "${KUBE_CONTEXT}" get \
    deployments,pods,services,persistentvolumeclaims,ingresses,horizontalpodautoscalers \
    --namespace "${APP_NAMESPACE}"

  log "Мониторинг"
  kubectl --context "${KUBE_CONTEXT}" get pods,services --namespace "${MONITORING_NAMESPACE}"
}

stop_dependencies() {
  # Deployment сохраняются в кластере, чтобы их PVC не удалялись вместе с локальными данными.
  local deployment
  for deployment in postgresql minio rabbitmq opensearch jaeger; do
    if kubectl --context "${KUBE_CONTEXT}" get "deployment/${deployment}" \
      --namespace "${APP_NAMESPACE}" >/dev/null 2>&1; then
      kubectl --context "${KUBE_CONTEXT}" scale "deployment/${deployment}" \
        --namespace "${APP_NAMESPACE}" \
        --replicas=0
    fi
  done
}

bring_up() {
  require_base_commands
  require_helm_3
  check_cluster
  build_image
  ensure_namespace "${APP_NAMESPACE}"
  # Порядок важен: зависимости -> CRD и мониторинг -> миграции и компоненты приложения.
  deploy_dependencies
  deploy_monitoring
  deploy_application
  show_status

  log "Локальное Kubernetes-окружение готово"
}

bring_down() {
  require_base_commands
  check_cluster

  log "Удаление Helm-релиза GophProfile"
  "${HELM_BIN}" --kube-context "${KUBE_CONTEXT}" uninstall "${APP_RELEASE}" \
    --namespace "${APP_NAMESPACE}" \
    --ignore-not-found

  log "Удаление Helm-релиза мониторинга"
  "${HELM_BIN}" --kube-context "${KUBE_CONTEXT}" uninstall "${MONITORING_RELEASE}" \
    --namespace "${MONITORING_NAMESPACE}" \
    --ignore-not-found

  log "Остановка локальных зависимостей с сохранением PVC"
  stop_dependencies
}

# Первый аргумент задает действие, которое должен выполнить скрипт.
case "${1:-}" in
  up)
    bring_up
    ;;
  down)
    bring_down
    ;;
  status)
    show_status
    ;;
  *)
    usage
    exit 1
    ;;
esac
