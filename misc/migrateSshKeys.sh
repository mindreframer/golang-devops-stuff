#!/usr/bin/env bash

##
# Ubuntu SSH system keys migration utility
#
# This utility enables the system-wide cloning of SSH keys located at /etc/ssh, ~/.ssh, and /root/.ssh from one host to
# another.  This makes it possible to migrate a hostname from one machine to another without triggering SSH key mismatch
# alerts.
#
# @author Jay Taylor [@jtaylor]
#
# @date 2013-09-03
#

function abortWithError() {
    echo "$1" 1>&2 && exit 1
}

test "$1" = '-h' && abortWithError "usage: $0 [source-ssh-host] [destination-ssh-host]"

test -z "$1" && abortWithError 'error: missing required parameter: [source-ssh-host] (see help with "-h" flag for usage details)'
sourceSshHost=$1
test -z "$2" && abortWithError 'error: missing required parameter: [destination-ssh-host] (see help with "-h" flag for usage details)'
destinationSshHost=$2

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

verifySshAndSudoForHosts "${sourceSshHost} ${destinationSshHost}"

# Copy root keys inside ssh users .ssh folder.
ssh -o 'BatchMode yes' -o 'StrictHostKeyChecking no' -o 'ConnectTimeout 15' -q $sourceSshHost \
'sudo rm -rf ~/.ssh/root ~/.ssh/etc
sudo cp -a /root/.ssh ~/.ssh/root
sudo chown -R ubuntu:ubuntu ~/.ssh/root
sudo cp -a /etc/ssh ~/.ssh/etc
sudo chown -R ubuntu:ubuntu ~/.ssh'

# I wish SSH had an FXP option like FTP, but it doesn't.
# So shuttle the files from A to B via this local system.
rm -rf /tmp/migrateSshKeys
rsync -azve "ssh -o 'BatchMode yes' -o 'StrictHostKeyChecking no' -o 'ConnectTimeout 15' -q" "${sourceSshHost}:~/.ssh" /tmp/migrateSshKeys
rsync -azve "ssh -o 'BatchMode yes' -o 'StrictHostKeyChecking no' -o 'ConnectTimeout 15' -q" /tmp/migrateSshKeys "${destinationSshHost}:~/"
rm -rf /tmp/migrateSshKeys

ssh -o 'BatchMode yes' -o 'StrictHostKeyChecking no' -o 'ConnectTimeout 15' -q $sourceSshHost 'rm -rf ~/.ssh/root ~/.ssh/etc'
ssh -o 'BatchMode yes' -o 'StrictHostKeyChecking no' -o 'ConnectTimeout 15' -q $destinationSshHost \
'ts=$(date +%Y%m%d_%H%M%S)
mv ~/.ssh{,.bak-$ts}
mv ~/{migrateSshKeys/,}.ssh
rmdir ~/migrateSshKeys
sudo chown -R root:root ~/.ssh/root
sudo test -d "/root/.ssh" && sudo mv /root/.ssh{,.bak-$ts}
sudo mv ~/.ssh/root /root/.ssh
sudo mv /etc/ssh{,.bak-$ts}
sudo mv ~/.ssh/etc /etc/ssh
sudo chmod 600 /etc/ssh/*key
sudo chown -R root:root /etc/ssh /root/.ssh
sudo service ssh restart'

