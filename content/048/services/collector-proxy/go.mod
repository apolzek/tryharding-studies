module github.com/obs-saas/collector-proxy

go 1.22

require github.com/obs-saas/shared v0.0.0

require github.com/golang-jwt/jwt/v5 v5.2.1 // indirect

replace github.com/obs-saas/shared => ../shared
