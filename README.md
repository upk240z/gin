# GIN API Server

## Settings
* Create directories
  ```
  mkdir tmp log
  ```
* Build
  ```
  go build
  ```
* CORS Origins in .env
  ```
  CORS_ORIGINS=http://xxxxx.xxx/,http://xxxxx.xxx/
  ```

## Start
  ```
  GIN_MODE=release ./gin-api --port 8888 &
  ```
