function abortIfNonZero() {
    # @param $1 command return code/exit status (e.g. $?, '0', '1').
    # @param $2 error message if exit status was non-zero.
    local rc=$1
    local what=$2
    test $rc -ne 0 && echo "error: ${what} exited with non-zero status ${rc}" 1>&2 && exit $rc || :
}

function abortWithError() {
    echo "$1" 1>&2 && exit 1
}

function warnIfNonZero() {
    # @param $1 command return code/exit status (e.g. $?, '0', '1').
    # @param $2 error message if exit status was non-zero.
    local rc=$1
    local what=$2
    test $rc -ne 0 && echo "warn: ${what} exited with non-zero status ${rc}" 1>&2 || :
}

function autoDetectServer() {
    # Attempts to auto-detect the server host by reading the contents of ../env/SB_SSH_HOST.
    if test -r '../env/SB_SSH_HOST'; then
        sbHost=$(head -n1 ../env/SB_SSH_HOST)
        test -n "${sbHost}" && echo "info: auto-detected shipbuilder host: ${sbHost}"
    else
        echo 'warn: server auto-detection failed: no such file: ../env/SB_SSH_HOST' 1>&2
    fi
}

function autoDetectFilesystem() {
    # Attempts to auto-detect the target filesystem type by reading the contents of ../env/LXC_FS.
    if test -r '../env/LXC_FS'; then
        lxcFs=$(head -n1 ../env/LXC_FS)
        test -n "${lxcFs}" && echo "info: auto-detected lxc filesystem: ${lxcFs}"
    else
        echo 'warn: lxc filesystem auto-detection failed: no such file: ../env/LXC_FS' 1>&2
    fi
}

function autoDetectZfsPool() {
    # When fs type is 'zfs', attempt to auto-detect the zfs pool name to create by reading the contents of ../env/ZFS_POOL.
    test -z "${lxcFs}" && autoDetectFilesystem # Attempt to ensure that the target filesystem type is available.
    if test "${lxcFs}" = 'zfs'; then
        if test -r '../env/ZFS_POOL'; then
            zfsPool=$(head -n1 ../env/ZFS_POOL)
            test -n "${zfsPool}" && echo "info: auto-detected zfs pool: ${zfsPool}"
            # Validate to ensure zfs pool name won't conflict with typical ubuntu root-fs items.
            for x in bin boot dev etc git home lib lib64 media mnt opt proc root run sbin selinux srv sys tmp usr var vmlinuz zfs-kstat; do
                test "${zfsPool}" = "${x}" && echo "error: invalid zfs pool name detected, '${x}' is a forbidden because it may conflict with a system directory" 1>&2 && exit 1
            done
        else
            echo 'warn: zfs pool auto-detection failed: no such file: ../env/ZFS_POOL' 1>&2
        fi
    fi
}

function verifySshAndSudoForHosts() {
    # @param $1 string. List of space-delimited SSH connection strings.
    local sshHosts="$1"
    echo "info: verifying ssh and sudo access for $(echo "${sshHosts}" | tr ' ' '\n' | grep -v '^ *$' | wc -l | sed 's/^[ \t]*//g') hosts"
    for sshHost in $(echo "${sshHosts}"); do
        echo -n "info:     testing host ${sshHost} .. "
        result=$(ssh -o 'BatchMode yes' -o 'StrictHostKeyChecking no' -o 'ConnectTimeout 15' -q $sshHost 'sudo -n echo "succeeded" 2>/dev/null')
        rc=$?
        test $rc -ne 0 && echo 'failed' && abortWithError "error: ssh connection test failed for host: ${sshHost} (exited with status code: ${rc})"
        test -z "${result}" && echo 'failed' && abortWithError "error: sudo access test failed for host: ${sshHost}"
        echo 'succeeded'
    done
}

function initSbServerKeys() {
    # @precondition $sbHost must not be empty.
    test -z "${sbHost}" && echo 'error: initSbServerKeys(): required parameter $sbHost cannot be empty' 1>&2 && exit 1
    echo "info: checking SB server \"${sbHost}\" SSH keys, will generate if missing"

    ssh -o 'BatchMode yes' -o 'StrictHostKeyChecking no' $sbHost '/bin/bash -c '"'"'
    echo "remote: info: setting up pub/private SSH keys so that root and main users can SSH in to either account"
    function abortIfNonZero() {
        local rc=$1
        local what=$2
        test $rc -ne 0 && echo "remote: error: ${what} exited with non-zero status ${rc}" && exit $rc || :
    }
    if ! test -e ~/.ssh/id_rsa.pub; then
        echo "remote: info: generating a new private/public key-pair for main user"
        rm -f ~/.ssh/id_*
        abortIfNonZero $? "removing old keys failed"
        ssh-keygen -f ~/.ssh/id_rsa -t rsa -N ""
        abortIfNonZero $? "ssh-keygen command failed"
    fi
    if test -z "$(grep "$(cat ~/.ssh/id_rsa.pub)" ~/.ssh/authorized_keys)"; then
        echo "remote: info: adding main user to main user authorized_keys"
        cat ~/.ssh/id_rsa.pub >> ~/.ssh/authorized_keys
        abortIfNonZero $? "appending public-key to authorized_keys command"
        chmod 600 ~/.ssh/authorized_keys
        abortIfNonZero $? "chmod 600 ~/.ssh/authorized_keys command"
    fi
    if sudo test -z "$(sudo grep "$(cat ~/.ssh/id_rsa.pub)" /root/.ssh/authorized_keys)"; then
        echo "remote: info: adding main user to root user authorized_keys"
        cat ~/.ssh/id_rsa.pub | sudo tee -a /root/.ssh/authorized_keys >/dev/null
        abortIfNonZero $? "appending public-key to authorized_keys command"
        sudo chmod 600 /root/.ssh/authorized_keys
        abortIfNonZero $? "chmod 600 /root/.ssh/authorized_keys command"
    fi

    if ! sudo test -e /root/.ssh/id_rsa.pub; then
        echo "remote: info: generating a new private/public key-pair for root user"
        sudo rm -f /root/.ssh/id_*
        abortIfNonZero $? "removing old keys failed"
        sudo ssh-keygen -f /root/.ssh/id_rsa -t rsa -N ""
        abortIfNonZero $? "ssh-keygen command failed"
    fi
    if sudo test -z "$(sudo grep "$(sudo cat /root/.ssh/id_rsa.pub)" /root/.ssh/authorized_keys)"; then
        echo "remote: info: adding root to root user authorized_keys"
        sudo cat /root/.ssh/id_rsa.pub | sudo tee -a /root/.ssh/authorized_keys >/dev/null
        abortIfNonZero $? "appending public-key to authorized_keys command"
        sudo chmod 600 /root/.ssh/authorized_keys
        abortIfNonZero $? "chmod 600 /root/.ssh/authorized_keys command"
    fi
    if test -z "$(grep "$(sudo cat /root/.ssh/id_rsa.pub)" ~/.ssh/authorized_keys)"; then
        echo "remote: info: adding root to main user authorized_keys"
        sudo cat /root/.ssh/id_rsa.pub >> ~/.ssh/authorized_keys
        abortIfNonZero $? "appending public-key to authorized_keys command"
        sudo chmod 600 ~/.ssh/authorized_keys
        abortIfNonZero $? "chmod 600 ~/.ssh/authorized_keys command"
    fi'"'"
    abortIfNonZero $? 'ssh key initialization'
    echo 'info: ssh key initialization succeeded'
}

function getSbServerPublicKeys() {
    test -z "${sbHost}" && echo 'error: getSbServerPublicKeys(): missing required parameter: SSH hostname' 1>&2 && exit 1

    initSbServerKeys

    echo "info: retrieving public-keys from shipbuilder server: ${sbHost}"

    local pubKeys=$(ssh -o 'BatchMode yes' -o 'StrictHostKeyChecking no' $sbHost 'cat ~/.ssh/id_rsa.pub && echo "." && sudo cat /root/.ssh/id_rsa.pub')
    abortIfNonZero $? 'SSH public-key retrieval failed'

    unprivilegedPubKey=$(echo "${pubKeys}" | grep --before 100 '^\.$' | grep -v '^\.$')
    rootPubKey=$(echo "${pubKeys}" | grep --after 100 '^\.$' | grep -v '^\.$')

    if [ -z "${unprivilegedPubKey}" ]; then
        echo 'error: failed to obtain build-server public-key for unprivileged user' 1>&2
        exit 1
    fi
    echo "info: obtained unprivileged public-key: ${unprivilegedPubKey}"
    if [ -z "${rootPubKey}" ]; then
        echo 'error: failed to obtain build-server public-key for root user' 1>&2
        exit 1
    fi
    echo "info: obtained root public-key: ${rootPubKey}"
}

function installAccessForSshHost() {
    # @precondition Variable $sshKeysCommand must be initialized and not empty.
    # @param $1 SSH connection string (e.g. user@host)
    local sshHost=$1

    if [ -z "${unprivilegedPubKey}" ] || [ -z "${rootPubKey}" ]; then
        getSbServerPublicKeys
    fi

    test -z "${sshHost}" && echo 'error: installAccessForSshHost(): missing required parameter: SSH hostname' 1>&2 && exit 1

    echo "info: setting up remote access from build-server to host: ${sshHost}"
    ssh -o 'BatchMode yes' -o 'StrictHostKeyChecking no' $sshHost '/bin/bash -c '"'"'
    function abortIfNonZero() {
        local rc=$1
        local what=$2
        test $rc -ne 0 && echo "remote: error: ${what} exited with non-zero status ${rc}" && exit $rc || :
    }
    echo "remote: checking main user.."
    if test -z "$(grep "'"${unprivilegedPubKey}"'" ~/.ssh/authorized_keys)" || test -z "$(sudo grep "'"${rootPubKey}"'" ~/.ssh/authorized_keys)"; then
        echo -e "'"${unprivilegedPubKey}\n${rootPubKey}"'" >> ~/.ssh/authorized_keys
        abortIfNonZero $? "appending public-keys to authorized_keys command"
        chmod 600 ~/.ssh/authorized_keys
        abortIfNonZero $? "chmod 600 ~/.ssh/authorized_keys command"
    fi
    echo "remote: checking root user.."
    if sudo test -z "$(sudo grep "'"${unprivilegedPubKey}"'" /root/.ssh/authorized_keys)" || sudo test -z "$(sudo grep "'"${rootPubKey}"'" /root/.ssh/authorized_keys)"; then
        echo -e "'"${unprivilegedPubKey}\n${rootPubKey}"'" | sudo tee -a /root/.ssh/authorized_keys >/dev/null
        abortIfNonZero $? "appending public-keys to authorized_keys command"
        sudo chmod 600 /root/.ssh/authorized_keys
        abortIfNonZero $? "chmod 600 /root/.ssh/authorized_keys command"
    fi
    exit 0
    '"'"
    abortIfNonZero $? "ssh access installation failed for host ${sshHost}"
    echo 'info: ssh access installation succeeded'
}

function installLxc() {
    # @param $1 $lxcFs lxc filesystem to use (zfs, btrfs are both supported).
    local lxcFs=$1
    test -z "${lxcFs}" && echo 'error: installLxc() missing required parameter: $lxcFs' 1>&2 && exit 1
    echo 'info: a supported version of lxc must be installed (as of 2013-07-02, `buntu comes with 0.7.x by default, we require is 0.9.0 or greater)'
    echo 'info: adding lxc daily ppa'
    sudo apt-add-repository -y ppa:ubuntu-lxc/daily
    abortIfNonZero $? "command 'sudo apt-add-repository -y ppa:ubuntu-lxc/daily'"
    sudo apt-get update
    abortIfNonZero $? "command 'sudo add-get update'"
    sudo apt-get install -y lxc lxc-templates
    abortIfNonZero $? "command 'apt-get install -y ${required}'"

    echo "info: installed version $(lxc-version) (should be >= 0.9.0)"

    # Add supporting package(s) for selected filesystem type.
    local fsPackages="$(test "${lxcFs}" = 'btrfs' && echo 'btrfs-tools' || :) $(test "${lxcFs}" = 'zfs' && echo 'zfs-fuse' || :)"

    local required="${fsPackages} git mercurial bzr build-essential bzip2 daemontools ntp ntpdate"
    echo "info: installing required build-server packages: ${required}"
    sudo apt-get install -y $required
    abortIfNonZero $? "command 'apt-get install -y ${required}'"

    local recommended='aptitude htop iotop unzip screen bzip2 bmon'
    echo "info: installing recommended packages: ${recommended}"
    sudo apt-get install -y $recommended
    abortIfNonZero $? "command 'apt-get install -y ${recommended}'"
    echo 'info: installLxc() succeeded'
}

function prepareNode() {
    # @param $1 $device to format and use for new mount.
    # @param $2 $lxcFs lxc filesystem to use (zfs, btrfs are both supported).
    # @param $3 $swapDevice to format and use as for swap (optional).
    # @param $4 $zfsPool zfs pool name to create (only required when lxc filesystem is zfs).
    local device=$1
    local lxcFs=$2
    local swapDevice=$3
    local zfsPool=$4
    test "${device}" = "${swapDevice}" && echo 'error: prepareNode() device & swapDevice must be different' 1>&2 && exit 1
    test -z "${device}" && echo 'error: prepareNode() missing required parameter: $device' 1>&2 && exit 1
    test ! -e "${device}" && echo "error: unrecognized device '${device}'" 1>&2 && exit 1
    test -z "${lxcFs}" && echo 'error: prepareNode() missing required parameter: $lxcFs' 1>&2 && exit 1
    test "${lxcFs}" = 'zfs' && test -z "${zfsPool}" && echo 'error: prepareNode() missing required zfs parameter: $zfsPool' 1>&2 && exit 1

    installLxc $lxcFs

    echo "info: attempting to unmount /mnt and ${device} to be safe"
    sudo umount /mnt 1>&2 2>/dev/null
    sudo umount $device 1>&2 2>/dev/null

    if ! [ -d /mnt/build ]; then
        echo 'info: creating /mnt/build mount point'
        sudo mkdir -p /mnt/build
        abortIfNonZero $? "creating /mnt/build"
    fi

    # Try to temporarily mount the device to get an accurate FS-type reading.
    sudo mount $device /mnt 1>&2 2>/dev/null
    fs=$(sudo df -T $device | tail -n1 | sed 's/[ \t]\+/ /g' | cut -d' ' -f2)
    test -z "${fs}" && echo "error: failed to determine FS type for ${device}" 1>&2 && exit 1
    sudo umount $device 1>&2 2>/dev/null

    echo "info: existing fs type on ${device} is ${fs}"
    if [ "${fs}" = "${lxcFs}" ]; then
        echo "info: ${device} is already formatted with ${lxcFs}"

    else
        # Purge any pre-existing fstab /mnt entries.
        sudo sed -i '/.*[ \t]\/mnt[ \t].*/d' /etc/fstab

        echo "info: formatting ${device} with ${lxcFs}"
        if test "${lxcFs}" = 'btrfs'; then
            sudo mkfs.btrfs $device
            abortIfNonZero $? "mkfs.btrfs ${device}"

            echo "info: updating /etc/fstab to map /mnt/build to the ${lxcFs} device"
            if [ -z "$(grep "$(echo $device | sed 's:/:\\/:g')" /etc/fstab)" ]; then
                echo "info: adding new fstab entry for ${device}"
                echo "${device} /mnt/build auto defaults 0 0" | sudo tee -a /etc/fstab >/dev/null
                abortIfNonZero $? "fstab add"
            else
                echo 'info: editing existing fstab entry'
                sudo sed -i "s-^\s*${device}\s\+.*\$-${device} /mnt/build auto defaults 0 0-" /etc/fstab
                abortIfNonZero $? "fstab edit"
            fi

            echo "info: mounting device ${device}"
            sudo mount $device
            abortIfNonZero $? "mounting ${device}"

        elif test "${lxcFs}" = 'zfs'; then
            # Create ZFS pool mount point.
            test ! -d "/${zfsPool}" && sudo rm -rf "/${zfsPool}" && sudo mkdir "/${zfsPool}" || :
            abortIfNonZero $? "creating /${zfsPool} mount point"

            # Create ZFS pool and attach to a device.
            if test -z "$(sudo zfs list -o name,mountpoint | sed '1d' | grep "^${zfsPool}.*\/${zfsPool}"'$')"; then
                # Format the device with any filesystem (mkfs.ext4 is fast).
                sudo mkfs.ext4 -q $device
                abortIfNonZero $? "command 'sudo mkfs.ext4 -q ${device}'"

                sudo zpool destroy $zfsPool 2>/dev/null

                sudo zpool create $zfsPool $device
                abortIfNonZero $? "command 'sudo zpool create -o ashift=12 ${zfsPool} ${device}'"
            fi

            # Create lxc and git volumes.
            for volume in lxc git; do
                test -z "$(sudo zfs list -o name | sed '1d' | grep "^${zfsPool}\/${volume}")" && sudo zfs create -o compression=on $zfsPool/$volume || :
                abortIfNonZero $? "command 'sudo zfs create -o compression=on ${zfsPool}/${volume}'"
            done

            # Unmount all ZFS volumes.
            sudo zfs umount -a
            abortIfNonZero $? "command 'sudo zfs umount -a'"

            # Export the pool.
            sudo zpool export $zfsPool
            abortIfNonZero $? "command 'sudo zpool export ${zfsPool}'"

            # Import the zfs pool, this will mount the volumes.
            sudo zpool import $zfsPool
            abortIfNonZero $? "command 'sudo zpool import ${zfsPool}'"

            # Enable zfs snapshot listing.
            sudo zpool set listsnapshots=on $zfsPool
            abortIfNonZero $? "command 'sudo zpool set listsnapshots=on ${zfsPool}'"

            # Add zfsroot to lxc configuration.
            test -z "$(sudo grep '^zfsroot=' /etc/lxc/lxc.conf 2>/dev/null)" && echo "zfsroot=${zfsPool}" | sudo tee -a /etc/lxc/lxc.conf || sudo sed -i "s/^zfsroot=.*/zfsroot=${zfsPool}/g" /etc/lxc/lxc.conf
            abortIfNonZero $? 'application of lxc zfsroot setting'

            # Remove any fstab entry for the ZFS device (ZFS will auto-mount one pool).
            sudo sed -i "/.*$(echo "${device}" | sed 's/\//\\\//g').*/d" /etc/fstab

            # Chmod 777 /$zfsPool/git
            sudo chmod 777 /$zfsPool/git
            abortIfNonZero $? "command 'sudo chmod 777 /${zfsPool}/git'"

            # Link /var/lib/lxc to /${zfsPool}/lxc, and then link /mnt/build/lxc to /var/lib/lxc.
            test -d '/var/lib/lxc' && sudo mv /var/lib/lxc{,.bak} || :
            test ! -h '/var/lib/lxc' && sudo ln -s /$zfsPool/lxc /var/lib/lxc || :
            test ! -h '/mnt/build/lxc' && sudo ln -s /$zfsPool/lxc /mnt/build/lxc || :

            # Also might as well resolve the git linkage while we're here.
            test ! -h '/mnt/build/git' && sudo ln -s /$zfsPool/git /mnt/build/git || :
            test ! -h '/git' && sudo ln -s /$zfsPool/git /git || :

        else
            echo "error: prepareNode() got unrecognized filesystem \"${lxcFs}\"" 1>&2
            exit 1
        fi
    fi

    if [ -d /var/lib/lxc ] && ! [ -e /mnt/build/lxc ]; then
        echo 'info: creating and linking /mnt/build/lxc folder'
        sudo mv /{var/lib,mnt/build}/lxc
        abortIfNonZero $? "lxc directory migration"
        sudo ln -s /mnt/build/lxc /var/lib/lxc
        abortIfNonZero $? "lxc directory symlink"
    fi

    if ! [ -d /mnt/build/lxc ]; then
        echo 'info: attempting to create missing /mnt/build/lxc'
        sudo mkdir /mnt/build/lxc
        abortIfNonZero $? "lxc directory creation"
    fi

    if ! [ -e /var/lib/lxc ]; then
        echo 'info: attemtping to symlink missing /var/lib/lxc to /mnt/build/lxc'
        sudo ln -s /mnt/build/lxc /var/lib/lxc
        abortIfNonZero $? "lxc directory symlink 2nd attempt"
    fi

    if ! test -z "${swapDevice}" && test -e "${swapDevice}"; then
        echo "info: activating swap device or partition: ${swapDevice}"
        # Purge any pre-existing fstab entries before adding the swap device.
        sudo sed -i "/^$(echo "${swapDevice}" | sed 's/\//\\&/g')[ \t].*/d" /etc/fstab
        abortIfNonZero $? "purging pre-existing ${swapDevice} entries from /etc/fstab"
        echo "${swapDevice} none swap sw 0 0" | sudo tee -a /etc/fstab 1>/dev/null
        sudo swapoff --all
        abortIfNonZero $? "adding ${swapDevice} to /etc/fstab"
        sudo mkswap -f $swapDevice
        abortIfNonZero $? "mkswap ${swapDevice}"
        sudo swapon --all
        abortIfNonZero $? "sudo swapon --all"
    fi

    # Install updated kernel if running Ubuntu 12.x series so lxc-attach will work.
    majorVersion=$(lsb_release --release | sed 's/^[^0-9]*\([0-9]*\)\..*$/\1/')
    if test $majorVersion -eq 12; then
        echo 'info: installing 3.8 or newer kernel, a system restart will be required to complete installation'
        sudo apt-get install -y linux-generic-lts-raring-eol-upgrade
        abortIfNonZero $? 'installing linux-generic-lts-raring-eol-upgrade'
        echo 1 | sudo tee -a /tmp/SB_RESTART_REQUIRED
    fi

    echo 'info: prepareNode() succeeded'
}

function prepareLoadBalancer() {
    # @param $1 ssl certificate base filename (without path).
    certFile=/tmp/$1

    version=$(lsb_release -a 2>/dev/null | grep "Release" | grep -o "[0-9\.]\+$")

    if [ "${version}" = "12.04" ]; then
        ppa=ppa:vbernat/haproxy-1.5
    elif [ "${version}" = "13.04" ]; then
        ppa=ppa:nilya/haproxy-1.5
    else
        echo "error: unrecognized version of ubuntu: ${version}" 1>&2 && exit 1
    fi
    echo "info: adding ppa repository for ${version}: ${ppa}"
    sudo apt-add-repository -y ${ppa}
    abortIfNonZero $? "adding apt repository ppa ${ppa}"

    required="haproxy"
    echo "info: installing required packages: ${required}"
    sudo apt-get update
    abortIfNonZero $? "updating apt"
    sudo apt-get install -y $required
    abortIfNonZero $? "apt-get install ${required}"

    optional="vim-haproxy"
    echo "info: installing optional packages: ${optional}"
    sudo apt-get install -y $optional
    abortIfNonZero $? "apt-get install ${optional}"

    if [ -r "${certFile}" ]; then
        if ! [ -d "/etc/haproxy/certs.d" ]; then
            sudo mkdir /etc/haproxy/certs.d 2>/dev/null
            abortIfNonZero $? "creating /etc/haproxy/certs.d directory"
        fi

        echo "info: installing ssl certificate to /etc/haproxy/certs.d"
        sudo mv $certFile /etc/haproxy/certs.d/
        abortIfNonZero $? "moving certificate to /etc/haproxy/certs.d"

        sudo chmod 400 /etc/haproxy/certs.d/$(echo $certFile | sed "s/^.*\/\(.*\)$/\1/")
        abortIfNonZero $? "chmod 400 /etc/haproxy/certs.d/<cert-file>"

        sudo chown -R haproxy:haproxy /etc/haproxy/certs.d
        abortIfNonZero $? "chown haproxy:haproxy /etc/haproxy/certs.d"

    else
        echo "warn: no certificate file was provided, ssl support will not be available" 1>&2
    fi

    echo "info: enabling the HAProxy system service in /etc/default/haproxy"
    sudo sed -i "s/ENABLED=0/ENABLED=1/" /etc/default/haproxy
    abortIfNonZero $? "enabling haproxy service in /dev/default/haproxy"
    echo 'info: prepareLoadBalancer() succeeded'
}

function installGo() {
    if [ -z "$(which go)" ]; then
        echo 'info: installing go 1.1 on the build-server'
        curl --location --silent https://launchpad.net/ubuntu/+archive/primary/+files/golang-src_1.1-1_amd64.deb > /tmp/golang-src_1.1-1_amd64.deb
        abortIfNonZero $? 'golang-src package download'
        curl --location --silent https://launchpad.net/ubuntu/+archive/primary/+files/golang-go_1.1-1_amd64.deb > /tmp/golang-go_1.1-1_amd64.deb
        abortIfNonZero $? 'golang-go package download'
        sudo dpkg -i /tmp/golang-src_1.1-1_amd64.deb /tmp/golang-go_1.1-1_amd64.deb
        abortIfNonZero $? 'golang packages installation'
        mkdir ~/go 2>/dev/null
        abortIfNonZero $ 'creating directory ~/go'
        echo 'info: adding $GOPATH to ~/.bashrc'
        echo 'export GOPATH=$HOME/go' >> ~/.bashrc
        abortIfNonZero $? 'adding $GOPATH to ~/.bashrc var exports'
    else
        echo 'info: go already appears to be installed, not going to force it'
    fi
}

function gitLinkage() {
    echo 'info: creating and linking /mnt/build/git -> /git'
    test ! -d '/mnt/build/git' && test ! -h '/mnt/build/git' && sudo mkdir -p /mnt/build/git || :
    test -d '/mnt/build/git' && sudo chown 777 /mnt/build/git || :
    test ! -h '/git' && sudo ln -s /mnt/build/git /git || :
    echo 'info: git linkage succeeded'
}

function buildEnv() {
    echo 'info: add build servers hostname configuration'
    test ! -d /mnt/build/env && sudo rm -rf /mnt/build/env && sudo mkdir -p /mnt/build/env || :
    abortIfNonZero $? "creating directory /mnt/build/env"
    echo "${sbHost}" | sudo tee /mnt/build/env/SB_SSH_HOST
    abortIfNonZero $? "appending shipbuilder host to /mnt/build/env/SB_SSH_HOST"
    echo 'info: env configuration succeeded'
}

function rsyslogLoggingListeners() {
    echo 'info: enabling rsyslog listeners'
    echo '# UDP syslog reception.
    $ModLoad imudp
    $UDPServerAddress 0.0.0.0
    $UDPServerRun 514

    # TCP syslog reception.
    $ModLoad imtcp
    $InputTCPServerRun 10514' | sudo tee /etc/rsyslog.d/49-haproxy.conf
    echo 'info: restarting rsyslog'
    sudo service rsyslog restart
    if [ -e /etc/rsyslog.d/haproxy.conf ]; then
        echo 'info: detected existing rsyslog haproxy configuration, will disable it'
        sudo mv /etc/rsyslog.d/haproxy.conf /etc/rsyslog.d-haproxy.conf.disabled
    fi
    echo 'info: rsyslog configuration succeeded'
}

function getContainerIp() {
    local container="$1"
    local allowedAttempts=60
    local i=0
    echo "info: getting container ip-address for name '${container}'"
    while [ $i -lt $allowedAttempts ]; do
        maybeIp=$(sudo lxc-ls --fancy | grep "^${container}[ \t]\+" | head -n1 | sed 's/[ \t]\+/ /g' | cut -d' ' -f3 | sed 's/[^0-9\.]*//g')
        # Verify that after a few seconds the ip hasn't changed.
        if [ -n "${maybeIp}" ]; then
            sleep 5
            ip=$(sudo lxc-ls --fancy | grep "^${container}[ \t]\+" | head -n1 | sed 's/[ \t]\+/ /g' | cut -d' ' -f3 | sed 's/[^0-9\.]*//g')
            if [ "${ip}" = "${maybeIp}" ]; then
                echo "info: ip-address verified, value=${ip}"
                break
            else
                echo "warn: ip-address not stable try1=${maybeIp} try2=${ip}"
            fi
            echo "info: found an ip=${ip}"
        fi
        i=$(($i+1))
        sleep 1
    done
    if [ -n "${ip}" ]; then
        echo "info: found ip=${ip} for ${container} container"
    else
        echo "error: obtaining ip-address for container '${container}' failed after ${allowedAttempts} attempts" 1>&2
        exit 1
    fi
}

function lxcInitBase() {
    # @param $1 lxc filesystem to use.
    local lxcFs=$1
    test -z "${lxcFs}" && echo 'error: lxcInitBase() missing required parameter: $lxcFs' 1>&2 && exit 1

    echo 'info: clear any pre-existing "base" container'
    sudo lxc-stop -k -n base 2>/dev/null
    sudo lxc-destroy -n base 2>/dev/null

    echo 'info: creating base lxc container'
    sudo lxc-create -n base -B $lxcFs $(test "${lxcFs}" = 'zfs' && echo '--zfsroot=tank' || :) -t ubuntu
    abortIfNonZero $? "lxc-create base"

    echo 'info: configuring base lxc container..'
    sudo lxc-start --daemon -n base
    abortIfNonZero $? "lxc-start base"

    getContainerIp base
}

function lxcConfigBase() {
    echo "info: adding shipbuilder server's public-key to authorized_keys file in base container"
    test ! -e "/mnt/build/lxc/base/rootfs/home/ubuntu/.ssh" && sudo mkdir /mnt/build/lxc/base/rootfs/home/ubuntu/.ssh || :
    abortIfNonZero $? "base container .ssh directory"

    sudo cp ~/.ssh/id_rsa.pub /mnt/build/lxc/base/rootfs/home/ubuntu/.ssh/authorized_keys
    abortIfNonZero $? "base container ssh authorized_keys"

    sudo chown -R ubuntu:ubuntu /mnt/build/lxc/base/rootfs/home/ubuntu/.ssh
    abortIfNonZero $? "chown -R ubuntu:ubuntu base container .ssh"

    sudo chmod 700 /mnt/build/lxc/base/rootfs/home/ubuntu/.ssh
    abortIfNonZero $? "chmod 700 base container .ssh"

    sudo chmod 600 /mnt/build/lxc/base/rootfs/home/ubuntu/.ssh/authorized_keys
    abortIfNonZero $? "chmod 600 base container .ssh/authorized_keys"

    echo 'info: adding the container "ubuntu" user to the sudoers list'
    echo 'ubuntu ALL=(ALL) NOPASSWD: ALL' | sudo tee -a /mnt/build/lxc/base/rootfs/etc/sudoers >/dev/null
    abortIfNonZero $? "adding 'ubuntu' to container sudoers"

    echo 'info: updating apt repositories in container'
    ssh -o 'StrictHostKeyChecking no' -o 'BatchMode yes' ubuntu@$ip "sudo apt-get update"
    abortIfNonZero $? "container apt-get update"

    packages='daemontools git-core curl unzip'
    echo "info: installing packages to base container: ${packages}"
    ssh -o 'StrictHostKeyChecking no' -o 'BatchMode yes' ubuntu@$ip "sudo apt-get install -y ${packages}"
    abortIfNonZero $? "container apt-get install ${packages}"

    echo 'info: stopping base container'
    sudo lxc-stop -k -n base
    abortIfNonZero $? "lxc-stop base"
    echo 'info: base container configuration succeeded'
}

function lxcConfigBuildPack() {
    # @param $1 base-container suffix (e.g. 'python').
    # @param $2 list of packages to install.
    # @param $3 customCommands command to evaluate over SSH.
    # @param $4 lxc filesystem to use.
    local container="base-$1"
    local packages="$2"
    local customCommands="$3"
    local lxcFs=$4
    test -z "${lxcFs}" && echo 'error: lxcConfigBuildPack() missing required parameter: $lxcFs' 1>&2 && exit 1
    echo "info: creating build-pack ${container} container"
    sudo lxc-clone -s -B $lxcFs -o base -n $container
    sudo lxc-start -d -n $container
    getContainerIp $container

    echo "info: installing packages to ${container} container: ${packages}"
    ssh -o 'StrictHostKeyChecking no' -o 'BatchMode yes' ubuntu@$ip "sudo apt-get install -y ${packages}"
    abortIfNonZero $? "[${container}] container apt-get install ${packages}"

    if [ -n "${customCommands}" ]; then
        echo "info: running customCommands: ${customCommands}"
        ssh -o 'StrictHostKeyChecking no' -o 'BatchMode yes' ubuntu@$ip "${customCommands}"
        abortIfNonZero $? "[${container}] container customCommands command /${customCommands}/"
    fi

    echo "info: stopping ${container} container"
    sudo lxc-stop -k -n $container
    abortIfNonZero $? "[${container}] lxc-stop"
    echo 'info: build-pack configuration succeeded'
}

function lxcConfigBuildPacks() {
    # @param $1 lxc filesystem to use.
    local lxcFs=$1
    test -z "${lxcFs}" && echo 'error: lxcConfigBuildPacks() missing required parameter: $lxcFs' 1>&2 && exit 1
    for buildPack in $(ls -1 /mnt/build/build-packs); do
        echo "info: initializing build-pack: ${buildPack}"
        packages="$(cat /mnt/build/build-packs/$buildPack/container-packages 2>/dev/null)"
        customCommands="$(cat /mnt/build/build-packs/$buildPack/container-custom-commands 2>/dev/null)"
        lxcConfigBuildPack "${buildPack}" "${packages}" "${customCommands}" "${lxcFs}"
        echo 'info: build-pack initialized succeeded'
    done
}

function prepareServerPart1() {
    # @param $1 ShipBuilder server hostname or ip-address.
    # @param $2 device to format and use for new mount.
    # @param $3 lxc filesystem to use.
    # @param $4 $swapDevice to format and use as for swap (optional).
    # @param $5 $zfsPool zfs pool name to create (only required when lxc filesystem is zfs).
    sbHost=$1
    device=$2
    lxcFs=$3
    local swapDevice=$4
    local zfsPool=$5
    test -z "${sbHost}" && echo 'error: prepareServerPart1(): missing required parameter: shipbuilder host' 1>&2 && exit 1
    test -z "${device}" && echo 'error: prepareServerPart1(): missing required parameter: device' 1>&2 && exit 1
    test -z "${lxcFs}" && echo 'error: prepareServerPart1(): missing required parameter: lxcFs' 1>&2 && exit 1
    test "${device}" = "${swapDevice}" && echo 'error: prepareServerPart1() device & swapDevice must be different' 1>&2 && exit 1
    test "${lxcFs}" = 'zfs' && test -z "${zfsPool}" && echo 'error: prepareServerPart1() missing required zfs parameter: $zfsPool' 1>&2 && exit 1

    prepareNode $device $lxcFs $swapDevice $zfsPool
    abortIfNonZero $? 'prepareNode() failed'

    installGo
    abortIfNonZero $? 'installGo() failed'

    gitLinkage
    abortIfNonZero $? 'gitLinkage() failed'

    buildEnv
    abortIfNonZero $? 'buildEnv() failed'

    rsyslogLoggingListeners
    abortIfNonZero $? 'rsyslogLoggingListeners() failed'

    echo 'info: prepareServerPart1() succeeded'
}

function prepareServerPart2() {
    # @param $1 lxc filesystem to use.
    local lxcFs=$1
    test -z "${lxcFs}" && echo 'error: prepareServerPart2() missing required parameter: $lxcFs' 1>&2 && exit 1

    lxcInitBase $lxcFs
    abortIfNonZero $? 'lxcInitBase() failed'

    lxcConfigBase $lxcFs
    abortIfNonZero $? 'lxcConfigBase() failed'

    lxcConfigBuildPacks $lxcFs
    abortIfNonZero $? 'lxcConfigBuildPacks() failed'

    echo 'info: prepareServerPart2() succeeded'
}

