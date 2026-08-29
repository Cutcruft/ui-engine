# Установка

## curl | sh (Рекомендуется)

```sh
curl -fsSL https://ui-engine.dev/install.sh | sh
# установит bin/ui-engine в /usr/local/bin
```

Скрипт сам скачает нужный бинарь для вашей ОС/архитектуры (darwin/linux, amd64/arm64) с GitHub Releases.

## npm

```sh
npm i -g @ui-engine/cli
# или
npx @ui-engine/cli --version
# или
yarn global add @ui-engine/cli
```

`package.json`:
```json
{
  "bin": { "ui-engine": "./bin/ui-engine" }
}
```

## go install

```sh
go install github.com/ui-engine/cli@latest
# бинарь в $GOPATH/bin/ui-engine
```

## Из исходников

```sh
git clone https://github.com/ui-engine/ui-engine
cd ui-engine
task build-cli  # -> bin/ui-engine
./bin/ui-engine --help
```

## Проверка

```sh
ui-engine version
# ui-engine 0.1.0
ui-engine --help
```

## Требования

- Go 1.21+
- Node 18+ (для TS и VitePress)
- task (go install github.com/go-task/task/v3/cmd/task@latest) — опционально

## Обновление

```sh
# curl
curl -fsSL https://ui-engine.dev/install.sh | sh

# npm
npm update -g @ui-engine/cli

# go
go install github.com/ui-engine/cli@latest
```
