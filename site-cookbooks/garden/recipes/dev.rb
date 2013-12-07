file "/etc/profile.d/gopath.sh" do
  content <<-EOF
export GOPATH=/go
export PATH=/go/bin:$PATH

function goto {
  local p
  local f

  for p in `echo $GOPATH | tr ':' '\n'`; do
    f=`find ${p}/src -maxdepth 3 -type d | grep ${1} | head -n 1`
    if [ -n "$f" ]; then
      cd $f
      return
    fi
  done
}

export GARDEN_TEST_ROOTFS=/opt/warden/rootfs
EOF
end