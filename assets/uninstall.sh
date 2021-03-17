sudo systemctl stop hello-bot
sudo systemctl disable hello-bot
sudo rm -f /etc/systemd/system/hello-bot.service
sudo systemctl daemon-reload
