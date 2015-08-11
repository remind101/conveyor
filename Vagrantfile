# -*- mode: ruby -*-
# vi: set ft=ruby :

Vagrant.configure(2) do |config|
  config.vm.box = "ubuntu/trusty64"
  #config.vm.provision "shell", path: "./bin/install_ansible"
  config.vm.provision "shell", inline: <<-SCRIPT
  sudo rm -rf /etc/ansible/playbook
  sudo mkdir /etc/ansible/playbook
  echo "127.0.0.1 ansible_connection=local" > /etc/ansible/hosts
  sudo chown -R vagrant:vagrant /etc/ansible
  SCRIPT
  config.vm.provision "file", source: "roles", destination: "/etc/ansible"
  config.vm.provision "file", source: "site.yml", destination: "/etc/ansible/playbook/site.yml"
  config.vm.provision "shell", path: "./bin/ansible", keep_color: true
end
