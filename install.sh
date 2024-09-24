#!/usr/bin/env bash

# shellcheck disable=SC2292
if [ -z "${BASH}" ]; then
    echo >&2 Please run the install with bash.
    exit 1
fi

aptdcon="$(command -v aptdcon || true)"
dpkg="$(command -v dpkg || true)"
pkexec="$(command -v pkexec || true)"

if [[ ! -x "${aptdcon}" && ! -x "${dpkg}" ]]; then
    echo >&2 Install only runs on Debian based distributions.
    exit 2
fi

set -euo pipefail

ldddns_preinstall(){
    
    avahi_install(){
        echo "Installing avahi-daemon..."
        sudo apt-get install avahi-daemon avahi-discover avahi-utils libnss-mdns mdns-scan -y
    }

    avahi_status(){
        if [ $(sudo systemctl is-active avahi-daemon) == "inactive" ]; then
            echo "WARNING: avahi-daemon is inactive! Please activated to make use of ldddns..."
        fi
    }

    # Check if avahi-daemon exists
    echo "Checking dependencies..."
    which avahi-daemon > /dev/null || avahi_install

    # Will show a warning if avahi-daemon is not active 
    avahi_status

}

ldddns_install() {
    tmpdir="$(mktemp --directory)"

    # Make a cleanup function
    cleanup() {
        rm --recursive --force -- "${tmpdir}"
    }
    trap cleanup EXIT

    echo -n "Finding latest package name..."

    uname="$(uname -s)"
    arch="$(dpkg --print-architecture)"

    package=$(curl --proto =https --fail --location --silent --show-error "https://github.com/arnested/ldddns/releases/latest/download/checksums.txt" | grep --ignore-case "${uname}" | grep "${arch}" | grep \.deb | awk '{print $2}')
    echo " ${package}"

    echo "Downloading ${package}..."
    curl --proto =https --fail --location --progress-bar --output "${tmpdir}/${package}" "https://github.com/arnested/ldddns/releases/latest/download/${package}"

    echo "Installing ${package}..."
    if [[ -x "${aptdcon}"  ]]; then
        yes | aptdcon --hide-terminal --install "${tmpdir}/${package}" > /dev/null
    elif [[ -x "${pkexec}"  ]]; then
        pkexec dpkg --install  "${tmpdir}/${package}"
    else
        sudo dpkg --install  "${tmpdir}/${package}"
    fi
}

ldddns_preinstall
ldddns_install
