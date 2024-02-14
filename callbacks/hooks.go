package callbacks

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/Datosystem/go_api_core/app"
	"github.com/Datosystem/go_api_core/controller"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func logError(hook app.Webhook, prefix string, err error) {
	log.Println("Webhook " + hook.TYPE + " " + hook.CONTEXT + ": " + err.Error())
}

func LoadWebhooks(db *gorm.DB) {
	var modelWebhooks []app.Webhook
	db.Raw(`
		IF (EXISTS(SELECT * FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_SCHEMA = 'dbo' AND TABLE_NAME = 'WEBHOOKS'))
		BEGIN
			SELECT * FROM WEBHOOKS WHERE TYPE IN ('AfterUpdate')
		END
	`).Scan(&modelWebhooks)
	for _, hook := range modelWebhooks {
		app.AddModelHook(hook.CONTEXT, hook.TYPE, func(db *gorm.DB) {
			c := db.Statement.Context.Value("gin").(*gin.Context)

			model := db.Statement.Model

			primaries := map[string]any{}
			if len(hook.QUERY_ARGS) != 0 {
				fields := controller.GetPrimaryFields(db.Statement.Schema.ModelType)
				for _, field := range fields {
					primaries[field] = db.Statement.ReflectValue.FieldByName(field).Interface()
				}
			}

			go func(hook app.Webhook) {
				controller.Recover(c)

				body := model
				if len(hook.QUERY_ARGS) != 0 {
					args := controller.QueryMapArgs{
						Primaries: primaries,
					}
					json.Unmarshal([]byte(hook.QUERY_ARGS), &args)
					args.Model = model

					err := controller.QueryMap(c, app.DB, &args, controller.QueryMapConfig{SkipValidation: true})
					if err != nil {
						logError(hook, "", err)
					}
					if args.Count != 1 {
						return
					}
				}

				client := &http.Client{}
				headers := http.Header{}
				headers.Set("Authorization", c.GetHeader("Authorization"))

				var reader io.Reader
				if hook.METHOD == http.MethodPost || hook.METHOD == http.MethodPatch || hook.METHOD == http.MethodPut {
					if hook.BODY != "" {
						reader = bytes.NewBufferString(hook.BODY)
					} else {
						data, err := json.Marshal(body)
						if err == nil {
							reader = bytes.NewBuffer(data)
						}
						headers.Set("Content-Type", "application/json")
					}
				}
				req, _ := http.NewRequest(hook.METHOD, hook.URL, reader)
				req.Header = headers

				res, err := client.Do(req)
				if err != nil {
					logError(hook, "call "+hook.URL+" error:", err)
					return
				}
				defer res.Body.Close()
			}(hook)
		})
	}

}
