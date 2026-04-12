//go:build !prod

package main

import "github.com/gin-gonic/gin"

// В режиме разработки статику раздаёт Vite (порт 3000)
// Go только API на порту 8181
func setupStatic(r *gin.Engine) {
	// ничего не делаем — фронтенд на отдельном vite dev server
}