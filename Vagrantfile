# -*- mode: ruby -*-
# vi: set ft=ruby :
 
Vagrant.configure(2) do |config|
  config.vm.box = "mitchellh/boot2docker"
  config.vm.network "forwarded_port", guest: 2375, host: 2377
  config.vm.network "forwarded_port", guest: 8080, host: 8080
  config.vm.provision :shell, inline: <<SCRIPT
/etc/init.d/docker stop
curl -L https://master.dockerproject.com/linux/amd64/docker-1.8.0-dev > /usr/local/bin/docker
chmod +x /usr/local/bin/docker
/etc/init.d/docker start
SCRIPT
end
