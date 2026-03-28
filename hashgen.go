package main 
import ("fmt";"golang.org/x/crypto/bcrypt") 
func main() { h,_:=bcrypt.GenerateFromPassword([]byte("Admin@1234"),10); fmt.Println(string(h)) } 
