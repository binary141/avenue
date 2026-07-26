# TODO
- [X] Get logout working
- [X] Only let the user interact with files that they have ownership of
- [] File permissions
- [] File sharing
- [X] Have an admin page to create and alter existing users
- [X] Allow files of any file size. Maybe make it env driven?
- [X] Folder creation in the ui
- [X] Breadcrumb the folder tree in the ui for easier navigation
- [X] Bulk delete of files (maybe have a multi-select in the list)
- [] Add way to move files around
- [] Fileviewer in ui? How would this be implemented?
- [] Password resets
- [] Add file search
- [] Add trash can
- [] Add worker to actually delete files that are in the trash after X days
- [] Consolidate api wrapper and have a unified logout if a 401 response is received. Currently broken in a handful of places

# backend

## ENV

All variables are optional; defaults are shown below.

### App

| Variable | Default | Description |
| --- | --- | --- |
| `APP_ENV` | `production` | Set to anything else (e.g. `dev`) to enable gin's debug logger and non-secure cookies. Only `production` marks session cookies `Secure` and puts gin in release mode. |
| `LOG_LEVEL` | `debug` | One of `debug`, `info`, `warn`, `error`. |
| `ALLOW_ORIGIN` | `http://localhost:5173` | Extra CORS origin allowed in addition to `http://localhost:8080` and `http://localhost:8081`. |
| `COOKIE_DOMAIN` | *(empty)* | `Domain` attribute for the session cookies. Leave unset to scope the cookie to whatever host serves the response. |
| `UPLOAD_DIR` | `./avenuectl/temp/` | Root directory (jailed) that uploaded file blobs are stored under. |

### Database

| Variable | Default | Description |
| --- | --- | --- |
| `DB_HOST` | `localhost` | |
| `DB_PORT` | `5432` | |
| `DB_USER` | `user` | |
| `DB_PASSWORD` | `secret` | |
| `DB_DATABASE` | `avenue` | |

### Root user seeding

| Variable | Default | Description |
| --- | --- | --- |
| `ROOT_USER_EMAIL` | `root@gmail.com` | Email for the seeded admin account, created on first boot if it doesn't exist. |
| `ROOT_USER_PASSWORD` | `password` | Password for the seeded admin account. |
| `ROOT_USER_RESET` | `false` | Set to `true` to reset the root user's password to `ROOT_USER_PASSWORD` on every boot. |

### Files & sharing

| Variable | Default | Description |
| --- | --- | --- |
| `MAX_FILE_BYTE_SIZE` | `209715200` (200MB) | Max upload size per file, in bytes. |
| `REGISTRATION_ENABLED` | `false` | Allows public self-registration via `/register` when `true`. |
| `ENABLE_FILE_SHARING` | `false` | Enables public share-link endpoints/routes for files. |
| `ENABLE_FOLDER_SHARING` | `false` | Enables public share-link endpoints/routes for folders. |

### Email (AWS SES)

| Variable | Default | Description |
| --- | --- | --- |
| `SES_FROM` | *(empty)* | Required to send email. The `From` address used for all outbound mail (password resets, etc). |
| `AWS_REGION` | | Standard AWS SDK env var, required alongside SES credentials. |
| `AWS_ACCESS_KEY_ID` / `AWS_SECRET_ACCESS_KEY` | | Standard AWS SDK credentials (or use an IAM role). |
| `GLOBAL_EMAIL_TO` | *(empty)* | When set, overrides the `To` address on every outbound email (useful for staging). |

### Sweepers

| Variable | Default | Description |
| --- | --- | --- |
| `TRASH_RETENTION` | `720h` (30 days) | How long an item sits in the trash before being permanently deleted. |
| `TRASH_SWEEP_INTERVAL` | `5m` | How often the trash sweeper runs. |
| `SESSION_SWEEP_INTERVAL` | `1h` | How often expired/invalidated sessions are purged from the database. |

# frontend

## ENV
```sh
VITE_APP_API_URL="http://localhost:8080/"
```

This template should help get you started developing with Vue 3 in Vite.

## Recommended IDE Setup

[VS Code](https://code.visualstudio.com/) + [Vue (Official)](https://marketplace.visualstudio.com/items?itemName=Vue.volar) (and disable Vetur).

## Recommended Browser Setup

- Chromium-based browsers (Chrome, Edge, Brave, etc.):
  - [Vue.js devtools](https://chromewebstore.google.com/detail/vuejs-devtools/nhdogjmejiglipccpnnnanhbledajbpd) 
  - [Turn on Custom Object Formatter in Chrome DevTools](http://bit.ly/object-formatters)
- Firefox:
  - [Vue.js devtools](https://addons.mozilla.org/en-US/firefox/addon/vue-js-devtools/)
  - [Turn on Custom Object Formatter in Firefox DevTools](https://fxdx.dev/firefox-devtools-custom-object-formatters/)

## Type Support for `.vue` Imports in TS

TypeScript cannot handle type information for `.vue` imports by default, so we replace the `tsc` CLI with `vue-tsc` for type checking. In editors, we need [Volar](https://marketplace.visualstudio.com/items?itemName=Vue.volar) to make the TypeScript language service aware of `.vue` types.

## Customize configuration

See [Vite Configuration Reference](https://vite.dev/config/).

## Project Setup

```sh
npm install
```

### Compile and Hot-Reload for Development

```sh
npm run dev
```

### Type-Check, Compile and Minify for Production

```sh
npm run build
```

### Lint with [ESLint](https://eslint.org/)

```sh
npm run lint
```
