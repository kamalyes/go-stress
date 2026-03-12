#!/bin/sh
# go-stress 一键安装脚本
#
# 用法:
#   curl -fsSL https://raw.githubusercontent.com/kamalyes/go-stress/master/install.sh | sh
#   wget -qO- https://raw.githubusercontent.com/kamalyes/go-stress/master/install.sh | sh
#
# 可用环境变量:
#   VERSION=v1.0.1          指定版本，默认取最新 Release
#   INSTALL_DIR=/opt/bin    指定安装目录，默认 /usr/local/bin（不可写时回落 ~/.local/bin）
#
# 行为: 检测系统与架构，从 GitHub Releases 下载对应预编译二进制并安装
set -eu

REPO="kamalyes/go-stress"
RELEASES_URL="https://github.com/${REPO}/releases"

# 下载先落临时目录，再移入安装目录
#
# 这一步是必须的：此前直接把下载目标设成 "${INSTALL_DIR}/go-stress.tmp"，
# 而 INSTALL_DIR 在「不可写但有 sudo」时仍会选 /usr/local/bin，于是不带 sudo
# 的下载必然失败（curl: (56) Failure writing output to destination）
# 更糟的是失败被 2>/dev/null 吞掉，脚本据此报「可能尚未发布」——
# 产物明明在 Release 里，报错方向完全错了
TMP_DIR="$(mktemp -d 2>/dev/null || mktemp -d -t gostress)"
cleanup() { rm -rf "$TMP_DIR"; }
trap cleanup EXIT INT TERM

die() {
  echo "❌ $1" >&2
  shift
  for line in "$@"; do echo "   $line" >&2; done
  exit 1
}

# ---------- 检测系统与架构 ----------
UNAME_S="$(uname -s)"
case "$(echo "$UNAME_S" | tr '[:upper:]' '[:lower:]')" in
  linux)   OS="linux" ;;
  darwin)  OS="darwin" ;;
  freebsd) OS="freebsd" ;;
  openbsd) OS="openbsd" ;;
  netbsd)  OS="netbsd" ;;
  dragonfly) OS="dragonfly" ;;
  sunos)   OS="solaris" ;;   # Solaris 的 uname -s 是 SunOS，而产物名为 solaris
  mingw*|msys*|cygwin*)
    die "请在 Windows 上直接下载 exe，而非通过 sh 安装" \
        "$RELEASES_URL" \
        "按架构选择 go-stress-windows-amd64.exe 或 go-stress-windows-arm64.exe"
    ;;
  *)
    die "暂不支持当前系统: $UNAME_S" "可到 $RELEASES_URL 查看已发布的产物"
    ;;
esac

UNAME_M="$(uname -m)"
case "$UNAME_M" in
  x86_64|amd64)   ARCH="amd64" ;;
  aarch64|arm64)  ARCH="arm64" ;;
  i386|i686)      ARCH="386" ;;
  ppc64le)        ARCH="ppc64le" ;;
  s390x)          ARCH="s390x" ;;
  mips64)         ARCH="mips64" ;;
  mips64le)       ARCH="mips64le" ;;
  *)
    die "暂不支持当前架构: $UNAME_M" \
        "已发布的架构见 $RELEASES_URL" \
        "本机有 Go 的话可改用: go install github.com/${REPO}@latest"
    ;;
esac

ASSET="go-stress-${OS}-${ARCH}"

# ---------- 选择安装目录 ----------
SUDO=""
if [ -n "${INSTALL_DIR:-}" ]; then
  mkdir -p "$INSTALL_DIR" 2>/dev/null || true
  [ -w "$INSTALL_DIR" ] || die "指定的 INSTALL_DIR 不可写: $INSTALL_DIR"
elif [ -w /usr/local/bin ]; then
  INSTALL_DIR="/usr/local/bin"
elif command -v sudo >/dev/null 2>&1 && [ -t 1 ]; then
  INSTALL_DIR="/usr/local/bin"
  SUDO="sudo"
  echo "🔐 /usr/local/bin 需要管理员权限，将使用 sudo 安装"
else
  # 管道执行且无交互终端时不要求 sudo：那样会在无人应答的密码提示上卡死
  INSTALL_DIR="${HOME}/.local/bin"
  mkdir -p "$INSTALL_DIR"
fi

BIN="${INSTALL_DIR}/go-stress"

# ---------- 组装下载地址 ----------
VERSION="${VERSION:-}"
if [ -n "$VERSION" ]; then
  DOWNLOAD_URL="${RELEASES_URL}/download/${VERSION}/${ASSET}"
  WHAT="${ASSET} (${VERSION})"
else
  DOWNLOAD_URL="${RELEASES_URL}/latest/download/${ASSET}"
  WHAT="${ASSET} (最新版)"
fi

# ---------- 下载 ----------
TMP_BIN="${TMP_DIR}/go-stress"
echo "⬇️  下载 ${WHAT} ..."

# 不吞错误输出：下载失败时要让人看见真实原因
if command -v curl >/dev/null 2>&1; then
  HTTP_CODE="$(curl -sSL -w '%{http_code}' -o "$TMP_BIN" "$DOWNLOAD_URL" || echo 000)"
elif command -v wget >/dev/null 2>&1; then
  if wget -q -O "$TMP_BIN" "$DOWNLOAD_URL"; then HTTP_CODE=200; else HTTP_CODE=000; fi
else
  die "需要 curl 或 wget，请先安装其一"
fi

case "$HTTP_CODE" in
  200) ;;
  404)
    die "Release 中没有 ${ASSET}（HTTP 404）" \
        "请确认该平台已发布: $RELEASES_URL" \
        "若指定了 VERSION，也请确认该版本存在"
    ;;
  000)
    die "下载失败，无法连接 github.com" \
        "网络不通或被拦截时，可到 $RELEASES_URL 手动下载" \
        "或用: go install github.com/${REPO}@latest"
    ;;
  *)
    die "下载失败，HTTP $HTTP_CODE" "地址: $DOWNLOAD_URL"
    ;;
esac

[ -s "$TMP_BIN" ] || die "下载到的文件为空" "地址: $DOWNLOAD_URL"

# 校验拿到的确实是可执行文件而非错误页
#
# GitHub 偶发以 200 返回 HTML 错误页，直接装上去会得到一个「能装上但跑不了」
# 的二进制，比装不上更难排查
if head -c 2 "$TMP_BIN" | grep -q '<' 2>/dev/null; then
  die "下载到的不是二进制而是网页内容" "地址: $DOWNLOAD_URL"
fi
chmod +x "$TMP_BIN"

# ---------- 安装 ----------
$SUDO mv -f "$TMP_BIN" "$BIN"
$SUDO chmod +x "$BIN"

# ---------- 验证 ----------
# 只取第一行：version 子命令会打印多行，整段插进来会把这条消息撑散
VERSION_LINE="$("$BIN" version 2>/dev/null | head -1 || true)"
[ -n "$VERSION_LINE" ] || die "安装到 $BIN 但无法执行" "请确认架构匹配: $UNAME_S/$UNAME_M"

echo ""
echo "✅ 安装完成: $BIN"
echo "   $VERSION_LINE"

case ":${PATH}:" in
  *":${INSTALL_DIR}:"*) ;;
  *)
    echo ""
    echo "⚠️  ${INSTALL_DIR} 不在 PATH 中，加入后方可直接用命令名调用:"
    echo "   echo 'export PATH=\"${INSTALL_DIR}:\$PATH\"' >> ~/.profile && . ~/.profile"
    ;;
esac

echo ""
echo "🚀 试一下:"
echo "   go-stress -url https://api.example.com -c 10 -n 100 -no-wait"
