#!/usr/bin/env bash

set -e

BINARY_NAME="harbinger"
INSTALL_DIR="/usr/local/bin"

# Function to handle installation
install_binary() {
    echo "Installing ${BINARY_NAME}..."
    
    OS="$(uname -s)"
    case ${OS} in
        Linux|Darwin)
            echo "Detected ${OS}. Installing to ${INSTALL_DIR}"
            if [ "${OS}" == "Linux" ]; then
                if [ -f "/proc/version" ]; then
                    PROC_VERSION=$(cat /proc/version)
                    if echo "${PROC_VERSION}" | grep -qi "microsoft"; then
                        if echo "${PROC_VERSION}" | grep -qi "WSL2"; then
                            echo "Running on WSL2."
                        else
                            echo "Running on WSL1."
                        fi
                    else
                        echo "Running on native Linux."
                    fi
                fi
            fi
            if [ ! -d "${INSTALL_DIR}" ]; then
                echo "Creating directory ${INSTALL_DIR}"
                mkdir -p "${INSTALL_DIR}"
            fi
            cp "./${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
            chmod +x "${INSTALL_DIR}/${BINARY_NAME}"
            echo "${BINARY_NAME} installed successfully to ${INSTALL_DIR}/${BINARY_NAME}"

            # If running on WSL, copy the PowerShell notification script
            if echo "${PROC_VERSION}" | grep -qi "microsoft"; then
                HARBINGER_CONFIG_DIR="${HOME}/.harbinger"
                mkdir -p "${HARBINGER_CONFIG_DIR}"
                cp "./scripts/windows/notify.ps1" "${HARBINGER_CONFIG_DIR}/notify.ps1"
                echo "PowerShell notification script copied to ${HARBINGER_CONFIG_DIR}/notify.ps1"
            fi
            ;;
        MINGW*|MSYS*|CYGWIN*)
            INSTALL_DIR_WIN="${HOME}/bin"
            echo "Detected Windows. Installing to ${INSTALL_DIR_WIN}"
            if [ ! -d "${INSTALL_DIR_WIN}" ]; then
                echo "Creating directory ${INSTALL_DIR_WIN}"
                mkdir -p "${INSTALL_DIR_WIN}"
            fi
            cp "./${BINARY_NAME}.exe" "${INSTALL_DIR_WIN}/${BINARY_NAME}.exe"
            echo "${BINARY_NAME} installed to ${INSTALL_DIR_WIN}/${BINARY_NAME}.exe"
            echo "Please ensure this directory is in your system's PATH."
            ;;
        *)
            echo "Unsupported operating system: ${OS}"
            exit 1
            ;;
    esac
}

# Function to handle uninstallation
uninstall_binary() {
    echo "Uninstalling ${BINARY_NAME}..."

    OS="$(uname -s)"
    case ${OS} in
        Linux|Darwin)
            if [ -f "${INSTALL_DIR}/${BINARY_NAME}" ]; then
                rm -f "${INSTALL_DIR}/${BINARY_NAME}"
                echo "${BINARY_NAME} uninstalled from ${INSTALL_DIR}/${BINARY_NAME}"
            else
                echo "${BINARY_NAME} not found in ${INSTALL_DIR}. Nothing to do."
            fi
            ;;
        MINGW*|MSYS*|CYGWIN*)
            INSTALL_DIR_WIN="${HOME}/bin"
            if [ -f "${INSTALL_DIR_WIN}/${BINARY_NAME}.exe" ]; then
                rm -f "${INSTALL_DIR_WIN}/${BINARY_NAME}.exe"
                echo "${BINARY_NAME} uninstalled from ${INSTALL_DIR_WIN}/${BINARY_NAME}.exe"
            else
                echo "${BINARY_NAME}.exe not found in ${INSTALL_DIR_WIN}. Nothing to do."
            fi
            ;;
        *)
            echo "Unsupported operating system: ${OS}"
            exit 1
            ;;
    esac
}

# Main script logic
if [ "$1" == "uninstall" ]; then
    uninstall_binary
else
    install_binary
fi