#!/bin/bash

# Exit immediately if a command exits with a non-zero status.
set -e

# --- Configuration ---
JAVA_VERSION_REQUIRED="17"
JDTLS_VERSION="1.39.0-202408291433"
JDTLS_URL="https://download.eclipse.org/jdtls/milestones/1.39.0/jdt-language-server-1.39.0-202408291433.tar.gz"
INSTALL_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/jdtls"
DOWNLOAD_DIR="${INSTALL_DIR}/download"
JDTLS_DIR="${INSTALL_DIR}/jdt-language-server-${JDTLS_VERSION}"

# --- Helper Functions ---
command_exists() {
    command -v "$1" >/dev/null 2>&1
}


# --- Java Installation ---
echo "--- Checking Java installation ---"
if command_exists java; then
    JAVA_VERSION_CURRENT=$(java -version 2>&1 | awk -F '"' '/version/ {print $2}' | awk -F. '{print $1}')
    if [ "$JAVA_VERSION_CURRENT" -ge "$JAVA_VERSION_REQUIRED" ]; then
        echo "Java version ${JAVA_VERSION_CURRENT} is already installed and meets the requirement (>= ${JAVA_VERSION_REQUIRED}). Skipping installation."
    else
        echo "Java version ${JAVA_VERSION_CURRENT} is installed but does not meet the requirement (>= ${JAVA_VERSION_REQUIRED}). Please upgrade Java manually.  # On macOS, you can use: brew install openjdk@17"
        # On macOS, you can use: brew install openjdk@17
        exit 1
    fi
else
    echo "Java is not installed. Please install Java ${JAVA_VERSION_REQUIRED} or higher.  # On macOS, you can use: brew install openjdk@17"
    # On macOS, you can use: brew install openjdk@17
    exit 1
fi



# --- JDT Language Server Installation ---
echo "--- Checking JDT Language Server installation ---"
if [ -d "${JDTLS_DIR}" ]; then
    echo "JDT Language Server appears to be installed in ${JDTLS_DIR}. Skipping installation."
else
    echo "JDT Language Server not found. Downloading and installing version ${JDTLS_VERSION}..."
    
    # Create installation directory if it doesn't exist
    mkdir -p "${DOWNLOAD_DIR}"
    mkdir -p "${JDTLS_DIR}"

    # Download JDTLS
    echo "Downloading from ${JDTLS_URL}..."
    tarball_name="jdt-language-server-${JDTLS_VERSION}.tar.gz"
    if command_exists wget; then
        wget -O "${DOWNLOAD_DIR}/${tarball_name}" "${JDTLS_URL}"
    elif command_exists curl; then
        curl -L -o "${DOWNLOAD_DIR}/${tarball_name}" "${JDTLS_URL}"
    else
        echo "Error: Neither wget nor curl is available. Please install one of them to proceed."
        exit 1
    fi
    
    # Extract JDTLS
    echo "Extracting to ${JDTLS_DIR}..."
    tar -xzf "${DOWNLOAD_DIR}/${tarball_name}" -C "${JDTLS_DIR}"
    
    # Clean up
    rm "${DOWNLOAD_DIR}/${tarball_name}"
    
    echo "JDT Language Server installed successfully in ${JDTLS_DIR}."
fi

# Set LAUNCHER_JAR environment variable
LAUNCHER_JAR=$(find "${JDTLS_DIR}/plugins" -name "org.eclipse.equinox.launcher_*.jar" | head -n 1)
if [ -z "$LAUNCHER_JAR" ]; then
    echo "Error: Could not find org.eclipse.equinox.launcher_*.jar in ${JDTLS_DIR}/plugins."
    exit 1
fi
export LAUNCHER_JAR
JDTLS_ROOT_PATH="${JDTLS_DIR}"
export JDTLS_ROOT_PATH

echo "--- Java environment setup complete! ---"