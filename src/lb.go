package main

import  (
  "fmt"
  "os/exec"
)

const (
  compute_node = "compute-10-2"
  port = "9090"
)


// boot the compute_resource
func RunCompute() {
  out, err := exec.Command("bash/runcompute.sh", compute_node, port).Output()
  if err != nil {
        fmt.Printf("%s", err)
    }
    fmt.Printf("%s", out)
}


// exit compute_resource
func KillCompute() {
  out, err := exec.Command("bash/killcompute.sh", compute_node).Output()
  if err != nil {
        fmt.Printf("%s", err)
    }
    fmt.Printf("%s", out)
}


func main() {

  RunCompute()

  // LoadBalancer()

  KillCompute()
}
