#!/bin/sh
set -e

REPO="liasica/cpm"
BINARY="cpm"
INSTALL_DIR="/usr/local/bin"

# 检测操作系统
detect_os() {
    case "$(uname -s)" in
        Darwin)  echo "darwin" ;;
        Linux)   echo "linux" ;;
        MINGW*|MSYS*|CYGWIN*) echo "windows" ;;
        *) echo "Unsupported OS: $(uname -s)" >&2; exit 1 ;;
    esac
}

# 检测架构
detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64)   echo "amd64" ;;
        aarch64|arm64)  echo "arm64" ;;
        *) echo "Unsupported architecture: $(uname -m)" >&2; exit 1 ;;
    esac
}

OS="$(detect_os)"
ARCH="$(detect_arch)"

echo "Detected: ${OS}/${ARCH}"

# 获取最新版本号
VERSION="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')"
if [ -z "$VERSION" ]; then
    echo "Failed to fetch latest version" >&2
    exit 1
fi

VERSION_NUM="${VERSION#v}"
echo "Latest version: ${VERSION}"

# 拼接下载地址
if [ "$OS" = "windows" ]; then
    FILENAME="${BINARY}_${VERSION_NUM}_${OS}_${ARCH}.zip"
else
    FILENAME="${BINARY}_${VERSION_NUM}_${OS}_${ARCH}.tar.gz"
fi

DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${FILENAME}"

# 创建临时目录
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

echo "Downloading ${DOWNLOAD_URL}..."
curl -fsSL "$DOWNLOAD_URL" -o "${TMP_DIR}/${FILENAME}"

# 解压
if [ "$OS" = "windows" ]; then
    unzip -q "${TMP_DIR}/${FILENAME}" -d "$TMP_DIR"
else
    tar -xzf "${TMP_DIR}/${FILENAME}" -C "$TMP_DIR"
fi

# 安装
if [ -w "$INSTALL_DIR" ]; then
    install -m 755 "${TMP_DIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
else
    echo "Installing to ${INSTALL_DIR} (requires sudo)..."
    sudo install -m 755 "${TMP_DIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
fi

echo "cpm ${VERSION} installed to ${INSTALL_DIR}/${BINARY}"
