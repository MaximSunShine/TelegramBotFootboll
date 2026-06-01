## План
- [ ] Подготовка install.sh
- [ ] Создание образа ВМ
- [ ] Настройка systemd-сервиса
- [ ] Документация развёртывания

## Быстрый старт
```bash
# Запуск новой ВМ из образа:
yc compute instance create --source-image-name TelegramBotFootboll-v1 --name bot-prod