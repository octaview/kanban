package docs

import "github.com/swaggo/swag"

// @title           Kanban Board API
// @version         1.0
// @description     API for managing Kanban boards, columns, tasks, and user collaboration
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.kanban-api.com/support
// @contact.email  support@kanban-api.com

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      localhost:8080
// @BasePath  /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token

// @tag.name Users
// @tag.description User management operations

// @tag.name Boards
// @tag.description Board management operations

// @tag.name Columns
// @tag.description Column management operations

// @tag.name Tasks
// @tag.description Task management operations

// @tag.name Labels
// @tag.description Label management operations

// @tag.name Board Sharing
// @tag.description Board sharing operations

// Register swagger info
func SwaggerInfo() *swag.Spec {
	return swag.Instance
}