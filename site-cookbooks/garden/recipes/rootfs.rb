root_fs_url = "http://cfstacks.s3.amazonaws.com/lucid64.dev.tgz"
root_fs_checksum = "b2633b2ab4964f91402bb2d889f2f12449a8b828"

src_filename = File.basename(root_fs_url)
src_filepath = "#{Chef::Config['file_cache_path']}/#{src_filename}"

remote_file src_filepath do
  source root_fs_url
  checksum root_fs_checksum
  owner "root"
  group "root"
  mode 0644
end

bash "extract rootfs" do
  cwd ::File.dirname(src_filepath)

  code <<-EOH
    mkdir -p /opt/warden/rootfs
    tar xzf #{src_filename} -C /opt/warden/rootfs
  EOH

  not_if { ::File.directory?("/opt/warden/rootfs") }
end
