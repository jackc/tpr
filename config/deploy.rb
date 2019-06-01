require 'mina/scp'

require_relative 'secrets'

# Put any custom mkdir's in here for when `mina setup` is ran.
task :setup do
  queue! %[mkdir -p "#{deploy_to}/#{shared_path}"]
  queue! %[chmod g+rx,u+rwx "#{deploy_to}/#{shared_path}"]

  queue! %[touch "#{deploy_to}/#{shared_path}/tpr.conf"]
  queue! %[chmod 640 "#{deploy_to}/#{shared_path}/tpr.conf"]
  queue  %[echo "-----> Be sure to edit '#{deploy_to}/#{shared_path}/tpr.conf'."]

  queue! %[mkdir -p "#{deploy_to}/tmp/uploads"]
  queue! %[chmod g+rx,u+rwx "#{deploy_to}/tmp/uploads"]
end

desc "Deploys the current version to the server."
task :deploy do
  to :before_hook do
    `rake clobber build`
    scp_upload("build/tpr-linux", "#{deploy_to}/tmp/uploads/tpr")
    scp_upload("build/assets", "#{deploy_to}/tmp/uploads/assets", recursively: true)
  end

  deploy do
    queue! %[mv "#{deploy_to}/tmp/uploads/tpr" "#{deploy_to}/$build_path/tpr"]
    queue! %[mv "#{deploy_to}/tmp/uploads/assets" "#{deploy_to}/$build_path/assets"]
    invoke :'deploy:cleanup'

    to :launch do
      queue %[systemctl stop tpr; true]
      queue %[systemctl start tpr]
    end
  end
end
