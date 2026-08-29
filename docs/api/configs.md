# Конфиги API

## App

`app.yaml` — `name`, `root`, `screensDir`, `theme`, `state`, `net`, `hooks`, `keys`.

## State

`state.yaml` — `type: string|int|bool|object|list`, `required`, `secret`, `default`, `items`, `fields`.

## Theme

`theme.yaml` — `active`, `tokens` (`colors/spacing/radius/typography/shadows/animations`), `themes` (light/dark/custom).

## Screen

`screen.yaml` — `screen`, `name`, `layout`/`root`, `children` с `component`, `props`, `bind`, `text`, `if`, `repeat`, `animate`, `on`.

## Net / Hooks / Keys

`net.yaml` — `url`, `events.client/server`, `hooks.yaml` — `onReady/onRoute`, `keys.yaml` — `bindings`.
