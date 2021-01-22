package authentication

import "os"
var jwtSecret = []byte{}


func init() {
	jwtSecret = []byte(os.Getenv("JWT_SECRET"))
}
