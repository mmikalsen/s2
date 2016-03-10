// Package for request stuct and utility methods
package request

type Request interface 
{
	id 		string
	ttl 	int 
} 

// Get package ID
func (r Request) GetId() string
{
	return r.id
}