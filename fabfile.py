from fabric.api import *
import requests

project = "crosby"
repository = "git@github.com:Bowery/" + project + ".git"
crosby_hosts = [
  'ubuntu@ec2-54-226-183-252.compute-1.amazonaws.com'
]
# env.key_filename = '/home/ubuntu/.ssh/id_aws'
env.key_filename = '/Users/steve/.ssh/bowery.pem'
env.password = 'java$cript'

def restart_crosby():
  with cd('/home/ubuntu/gocode/src/crosby'):
    run('git pull')
    with cd('server'):
      sudo('GOPATH=/home/ubuntu/gocode go get -d')
      sudo('GOPATH=/home/ubuntu/gocode go build')
      run('myth static/style.css static/out.css')

    sudo('cp -f crosby.conf /etc/init/crosby.conf')
    sudo('initctl reload-configuration')
    sudo('restart crosby')

def crosby():
  execute(restart_crosby, hosts=crosby_hosts)
