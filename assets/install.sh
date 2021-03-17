sudo cp ./hello-bot.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now hello-bot
sudo systemctl status hello-bot
