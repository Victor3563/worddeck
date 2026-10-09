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
if ! command -v docker >/dev/null; then
  if ! curl -fsSL https://get.docker.com | sh; then
    rm -f /etc/apt/sources.list.d/docker.list /etc/apt/sources.list.d/docker.sources
    apt-get update -q
    apt-get install -yq docker.io
  fi
fi
systemctl enable --now docker
docker --version

echo "== minikube, kubectl"
curl -fsSLo /tmp/minikube https://storage.googleapis.com/minikube/releases/latest/minikube-linux-amd64
install -m 0755 /tmp/minikube /usr/local/bin/minikube
KV=$(curl -fsSL https://dl.k8s.io/release/stable.txt)
curl -fsSLo /tmp/kubectl "https://dl.k8s.io/release/$KV/bin/linux/amd64/kubectl"
curl -fsSLo /tmp/kubectl.sha256 "https://dl.k8s.io/release/$KV/bin/linux/amd64/kubectl.sha256"
echo "$(cat /tmp/kubectl.sha256)  /tmp/kubectl" | sha256sum --check
install -m 0755 /tmp/kubectl /usr/local/bin/kubectl
minikube version --short
kubectl version --client | head -1

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

echo "== minikube start, images"
sudo -iu deploy bash -c '
set -euo pipefail
minikube start --driver=docker --container-runtime=docker --cpus=max --memory=2600
eval $(minikube docker-env)
cd ~/worddeck
docker build -q -t worddeck-backend:1.0.0 --build-arg VERSION=1.0.0 backend
docker build -q -t worddeck-frontend:1.0.0 frontend
docker pull -q postgres:16-alpine
docker images | grep -E "worddeck|postgres"
kubectl get nodes
'

echo "== done"
