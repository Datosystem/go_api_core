package message

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// 2** - Success

// 200
func Ok(c *gin.Context) Message {
	return &Msg{
		Message: GetPrinter(c).Sprintf("Task performed successfully"),
		Status:  http.StatusOK,
	}
}

// 3** - User error

// 307
func ItemFound(c *gin.Context) Message {
	return &Msg{
		Message: GetPrinter(c).Sprintf("Item found"),
		Status:  http.StatusTemporaryRedirect,
	}
}

// 4** - User error

// 400
func InvalidUrlParameter(c *gin.Context, parameter string) Message {
	return &Msg{
		Message: GetPrinter(c).Sprintf("Missing or invalid required %s field in URL", parameter),
		Status:  http.StatusBadRequest,
	}
}

// 403
func Forbidden(c *gin.Context) Message {
	return &Msg{
		Message: GetPrinter(c).Sprintf("Forbidden"),
		Status:  http.StatusForbidden,
	}
}

func InsufficientPermissions(c *gin.Context, permissions ...string) Message {
	return &Msg{
		Message: GetPrinter(c).Sprintf("Permissions %s are required to access this resource, please contact your administrator", strings.Join(permissions, ",")),
		Status:  http.StatusForbidden,
	}
}

func InsufficientPermissionsHasOne(c *gin.Context, permissions ...string) Message {
	return &Msg{
		Message: GetPrinter(c).Sprintf("Permissions %s are required to access this resource, please contact your administrator", strings.Join(permissions, ",")),
		Status:  http.StatusForbidden,
	}
}

func UnauthorizedRelations(c *gin.Context, relations ...string) Message {
	return &Msg{
		Message: GetPrinter(c).Sprintf("You do not have sufficient permissions to access the following relations: %s", strings.Join(relations, ",")),
		Status:  http.StatusForbidden,
	}
}

// 404
func ItemNotFound(c *gin.Context) Message {
	return &Msg{
		Message: GetPrinter(c).Sprintf("The requested item was not found"),
		Status:  http.StatusNotFound,
	}
}

func FileNotFound(c *gin.Context) Message {
	return &Msg{
		Message: GetPrinter(c).Sprintf("The requested file was not found"),
		Status:  http.StatusNotFound,
	}
}

func FolderNotFound(c *gin.Context) Message {
	return &Msg{
		Message: GetPrinter(c).Sprintf("The requested folder was not found"),
		Status:  http.StatusNotFound,
	}
}

// 409
func Conflict(c *gin.Context) Message {
	return &Msg{
		Message: GetPrinter(c).Sprintf("Conflict"),
		Status:  http.StatusConflict,
	}
}

func DeleteFailed(c *gin.Context, blockingRelations []string) Message {
	return &Msg{
		Message: GetPrinter(c).Sprintf("Cannot delete the requested resource because it belongs to the following relations.<br>%s", strings.Join(blockingRelations, "<br>")),
		Status:  http.StatusConflict,
	}
}

func DuplicateUnique(c *gin.Context, table, combination string) Message {
	return &Msg{
		Message: GetPrinter(c).Sprintf("The combination %s already exists for %s", combination, table),
		Status:  http.StatusConflict,
	}
}

func CannotDeleteSharedResource(c *gin.Context) Message {
	return &Msg{
		Message: GetPrinter(c).Sprintf("Cannot delete the shared resource as you aren't its owner"),
		// You cannot delete the shared resource because you are not the owner
		Status: http.StatusConflict,
	}
}

func ConflictingPaginationAndAggregation(c *gin.Context) Message {
	return &Msg{
		Message: GetPrinter(c).Sprintf("Pagination is not supported with aggregations"),
		Status:  http.StatusConflict,
	}
}

func ConflictingOrderByAndDistinct(c *gin.Context) Message {
	return &Msg{
		Message: GetPrinter(c).Sprintf("The Order field must be specified in the Select field when using Distinct"),
		Status:  http.StatusConflict,
	}
}

func ConflictingUsername(c *gin.Context) Message {
	return &Msg{
		Message: GetPrinter(c).Sprintf("The username is already registered"),
		Status:  http.StatusConflict,
	}
}

func ManualPagination(c *gin.Context) Message {
	return &Msg{
		Message: GetPrinter(c).Sprintf("To paginate this request you must specify the Order attribute manually via the 'ord' param"),
		Status:  http.StatusConflict,
	}
}

func MissingRequiredParameter(c *gin.Context, name, in string) Message {
	return &Msg{
		Message: GetPrinter(c).Sprintf("Missing required parameter %s in %s", name, in),
		Status:  http.StatusConflict,
	}
}

func MissingForeignKey(c *gin.Context, key, rel string) Message {
	return &Msg{
		Message: GetPrinter(c).Sprintf("Could not find the foreign key %s, required by the relation %s, in its parent object.", key, rel),
		Status:  http.StatusConflict,
	}
}

// 422
func Unprocessable(c *gin.Context) Message {
	return &Msg{
		Message: GetPrinter(c).Sprintf("The submitted request present invalid or incomplete data"),
		Status:  http.StatusUnprocessableEntity,
	}
}

func InvalidJSON(c *gin.Context) Message {
	return &Msg{
		Message: GetPrinter(c).Sprintf("Missing or invalid JSON body"),
		Status:  http.StatusUnprocessableEntity,
	}
}

func InvalidParamsJSON(c *gin.Context) Message {
	return &Msg{
		Message: GetPrinter(c).Sprintf("The supplied params JSON isn't syntactically valid"),
		Status:  http.StatusUnprocessableEntity,
	}
}

func InvalidParamsSyntax(c *gin.Context) Message {
	return &Msg{
		Message: GetPrinter(c).Sprintf("The request cannot be completed due to invalid params syntax"),
		Status:  http.StatusUnprocessableEntity,
	}
}

func InvalidField(c *gin.Context, field string) Message {
	return &Msg{
		Message: GetPrinter(c).Sprintf("The requested field %s could not be found", field),
		Status:  http.StatusUnprocessableEntity,
	}
}

func InvalidFieldValue(c *gin.Context, field, rules string, value interface{}) Message {
	return &Msg{
		Message: GetPrinter(c).Sprintf("The specified value %v for the field %s must respect these constraits %s", value, field, rules),
		Status:  http.StatusUnprocessableEntity,
	}
}

func InvalidFieldAlias(c *gin.Context, alias, field string) Message {
	return &Msg{
		Message: GetPrinter(c).Sprintf("The alias %s specified for the field %s is invalid", alias, field),
		Status:  http.StatusUnprocessableEntity,
	}
}

func InvalidParamOperator(c *gin.Context, operator string) Message {
	return &Msg{
		Message: GetPrinter(c).Sprintf("The params operator %s is not supported", operator),
		Status:  http.StatusUnprocessableEntity,
	}
}

func InvalidParamType(c *gin.Context, field string, correctType string) Message {
	return &Msg{
		Message: GetPrinter(c).Sprintf("The supplied field \"%s\" needs to be of type \"%s\"", field, correctType),
		Status:  http.StatusUnprocessableEntity,
	}
}

func InvalidRelation(c *gin.Context, table string) Message {
	return &Msg{
		Message: GetPrinter(c).Sprintf("Missing or invalid specified relation %s", table),
		Status:  http.StatusUnprocessableEntity,
	}
}

func InvalidRelations(c *gin.Context, relations ...string) Message {
	return &Msg{
		Message: GetPrinter(c).Sprintf("The request cannot be completed due to invalid specified relations: %s", strings.Join(relations, ",")),
		Status:  http.StatusUnprocessableEntity,
	}
}

func InvalidOrders(c *gin.Context, orders ...string) Message {
	return &Msg{
		Message: GetPrinter(c).Sprintf("The request cannot be completed due to invalid specified order by: %s", strings.Join(orders, ",")),
		Status:  http.StatusUnprocessableEntity,
	}
}

func DuplicateStructField(c *gin.Context, field string) Message {
	return &Msg{
		Message: GetPrinter(c).Sprintf("Duplicate struct field %s, use an alias to avoid this error (eg. field AS alias)", field),
		Status:  http.StatusUnprocessableEntity,
	}
}

func InvalidFieldRequired(c *gin.Context, name string) Message {
	return &Msg{
		Message: GetPrinter(c).Sprintf("The %s property is required", name),
		Status:  http.StatusUnprocessableEntity,
	}
}

func RowError(c *gin.Context, row int, message string) Message {
	return &Msg{
		Message: GetPrinter(c).Sprintf("Row %d:%s", row, message),
		Status:  http.StatusUnprocessableEntity,
	}
}

func DisplayNameNotSupported(c *gin.Context) Message {
	return &Msg{
		Message: GetPrinter(c).Sprintf("This resource doesn't support DISPLAY_NAME"),
		Status:  http.StatusUnprocessableEntity,
	}
}

// 5** - Server error

func InternalServerError(c *gin.Context) Message {
	return &Msg{
		Message: GetPrinter(c).Sprintf("An error has occurred"),
		Status:  http.StatusInternalServerError,
	}
}

func ExpectedSlice(c *gin.Context) Message {
	return &Msg{
		Message: GetPrinter(c).Sprintf("The supplied parameter isn't of type *[]models.*"),
		Status:  http.StatusInternalServerError,
	}
}

func UnsupportedParamType(c *gin.Context, parameter string) Message {
	return &Msg{
		Message: GetPrinter(c).Sprintf("The supplied parameter %s isn't supported yet. This is an error", parameter),
		Status:  http.StatusInternalServerError,
	}
}

// System messages

type SkipDelete struct{}

func (SkipDelete) Error() string {
	return "___SKIP_DELETE___"
}
