package "iptables"
package "quota"
package "rsync"
package "ruby1.9.1"
package "ruby1.9.1-dev"
package "git"
package "lsof"

if ["debian", "ubuntu"].include?(node["platform"])
  if node["kernel"]["release"].end_with? "virtual"
    package "linux-image-extra" do
      package_name "linux-image-extra-#{node['kernel']['release']}"
      action :install
    end
  end
end

package "apparmor" do
  action :remove
end

execute "remove all remnants of apparmor" do
  command "sudo dpkg --purge apparmor"
end

execute "build root directory" do
  cwd "/vagrant"

  command "make GOPATH=/go"
  action :run
end

directory "/opt/warden" do
  mode 0755
  action :create
end

directory "/opt/warden/containers" do
  mode 0755
  action :create
end
