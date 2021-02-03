package web

import (
	"bytes"
	"html/template"
	"io/ioutil"

	"github.com/gin-gonic/gin"

	"github.com/covergates/covergates/config"
	"github.com/covergates/covergates/web"
)

// HandleIndex return HTML
func HandleIndex(config *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		buffer := bytes.NewBuffer([]byte{})
		html := web.MustLookup("/index.html")
		t, _ := template.New("index").Parse(string(html))
		_ = t.Execute(buffer, config.Server.Base)
		data, _ := ioutil.ReadAll(buffer)
		c.Data(200, "text/html; charset=UTF-8", data)
	}
}
