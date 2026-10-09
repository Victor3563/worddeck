#!/bin/bash
# Подготовка чистого сервера Ubuntu к запуску в minikube. Запускать от root из клона репозитория:
#   bash worddeck/scripts/server-setup.sh
set -euo pipefail

SRC=$(cd "$(dirname "$0")/.." && pwd)

echo "== packages"
export DEBIAN_FRONTEND=noninteractive
apt-get update -q
apt-get install -yq ca-certificates curl git ufw

echo "== docker"
rm -f /etc/apt/sources.list.d/docker.list /etc/apt/sources.list.d/docker.sources
command -v docker >/dev/null || apt-get install -yq docker.io
systemctl enable --now docker
docker --version

echo "== minikube"
curl -fsSL --retry 3 --max-time 300 -o /tmp/minikube https://storage.googleapis.com/minikube/releases/latest/minikube-linux-amd64
install -m 0755 /tmp/minikube /usr/local/bin/minikube
minikube version --short

echo "== swap"
if ! swapon --show | grep -q /swapfile; then
  fallocate -l 2G /swapfile
  chmod 600 /swapfile
  mkswap /swapfile
  swapon /swapfile
  grep -q '^/swapfile' /etc/fstab || echo '/swapfile none swap sw 0 0' >> /etc/fstab
fi

echo "== user deploy"
id deploy >/dev/null 2>&1 || useradd -m -s /bin/bash deploy
usermod -aG docker deploy
install -d -m 700 -o deploy -g deploy /home/deploy/.ssh
if [ -f /root/.ssh/authorized_keys ]; then
  install -m 600 -o deploy -g deploy /root/.ssh/authorized_keys /home/deploy/.ssh/authorized_keys
fi

echo "== project"
rm -rf /home/deploy/worddeck
cp -a "$SRC" /home/deploy/worddeck
chown -R deploy:deploy /home/deploy/worddeck

echo "== firewall"
ufw allow 22/tcp
ufw allow 8080/tcp
ufw --force enable

echo "== minikube start"
sudo -iu deploy minikube start --driver=docker --container-runtime=docker --cpus=max --memory=2600 \
  --registry-mirror=https://mirror.gcr.io --registry-mirror=https://dockerhub.timeweb.cloud

echo "== kubectl"
KUBECTL=$(docker exec minikube sh -c 'ls -d /var/lib/minikube/binaries/*/kubectl | tail -1')
docker cp "minikube:$KUBECTL" /usr/local/bin/kubectl
chmod 0755 /usr/local/bin/kubectl
kubectl version --client | head -1

echo "== images"
sudo -iu deploy bash -c '
set -euo pipefail
eval $(minikube docker-env)
cd ~/worddeck
docker build -q -t worddeck-backend:1.0.0 --build-arg VERSION=1.0.0 backend
docker build -q -t worddeck-frontend:1.0.0 --build-arg NPM_REGISTRY=https://registry.npmmirror.com/ frontend
docker pull -q postgres:16-alpine
docker images | grep -E "worddeck|postgres"
kubectl get nodes
'

echo "== done"
