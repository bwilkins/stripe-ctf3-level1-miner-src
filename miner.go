package main

import (
  "io"
  "io/ioutil"
  "os"
  "os/exec"
  "time"
  "crypto/rand"
  "fmt"
  "crypto/sha1"
  "hash"
  "bytes"
  "regexp"
)

type Result struct {
  Success bool
  CommitHash []byte
  CommitBody string
}

func ResetState() (err error) {
  err = exec.Command("/usr/local/bin/git", "fetch", "origin", "master").Run()
  if err == nil {
    err = exec.Command("/usr/local/bin/git", "reset", "--hard", "origin/master").Run()
  }
  return
}

func EntryIsInLedger() bool {
  ledger, _ := ioutil.ReadFile("LEDGER.txt")
  match, _ := regexp.Match(`/user-zsrh7rcm: \d+/`, ledger)
  return match
}

func UpdateLedger() (err error) {
  if ! EntryIsInLedger() {
    contents, _ := ioutil.ReadFile("LEDGER.txt")
    new_contents := []byte(fmt.Sprintf("%s\n%s\n", bytes.TrimSpace(contents), "user-zsrh7rcm: 1"))
    err = ioutil.WriteFile("LEDGER.txt", new_contents, os.ModeAppend)
  } else {
    // err = exec.Command("/usr/bin/perl", "-i", "-pe", `'s/(user-zsrh7rcm: )(\d+)/$1 . ($2+1)/e'`, "LEDGER.txt").Run()
  }
  if err == nil {
    err = exec.Command("/usr/local/bin/git", "add", "LEDGER.txt").Run()
  }
  return
}

func GetDifficulty() []byte {
  difficulty, _ := ioutil.ReadFile("difficulty.txt")
  return bytes.TrimSpace(difficulty)
}

func GetTree() []byte {
  tree_cmd := exec.Command("git", "write-tree")
  tree, _ := tree_cmd.Output()
  return bytes.TrimSpace(tree)
}

func GetParent() []byte {
  parent_cmd := exec.Command("git", "rev-parse", "HEAD")
  parent, _ := parent_cmd.Output()
  return bytes.TrimSpace(parent)
}

func GetTime() int64 {
  return time.Now().Unix()
}

func GetNonce() []byte {
  b := make([]byte, 8)
  rand.Read(b)
  return b
}

func PrebuildBody(tree, parent []byte, time_v int64) (body string) {
  body_fmt := `tree %s
parent %s
author CTF user <me@example.com> %d +0000
committer CTF user <me@example.com> %d +0000

Give me a Gitcoin`

  fbody := fmt.Sprintf(body_fmt, tree, parent, time_v, time_v)
  body = fmt.Sprintf("%s\n\n%s", fbody, "%s")

  return
}

func Solve(difficulty []byte, body string) (bool, []byte, string) {
  var nonce, checksum, checksum_hex []byte
  var full_body, hash_body string
  var hasher hash.Hash

  nonce = GetNonce()
  full_body = fmt.Sprintf(body, nonce)
  hash_body = fmt.Sprintf("commit %d%s%s", len(full_body), []byte{0}, full_body)

  hasher = sha1.New()
  io.WriteString(hasher, hash_body)
  checksum = hasher.Sum(nil)
  checksum_hex = []byte(fmt.Sprintf("%x", checksum))

  if bytes.Compare(checksum_hex, difficulty) == -1 {
    return true, checksum_hex, full_body
  }
  return false, checksum_hex, full_body
}

func Solver(difficulty []byte, body string, talkback chan Result) {
  var res Result
  for ;; {
   res.Success, res.CommitHash, res.CommitBody = Solve(difficulty, body)
   talkback <- res
  }
}

func GetGitCoin(result Result) bool {
  err := ioutil.WriteFile("tmpledger", []byte(result.CommitBody), 0600)
  hash_cmd := exec.Command("/usr/local/bin/git", "hash-object", "-t", "commit", "-w", "tmpledger")
  err = hash_cmd.Run()

  if err == nil {
    err = exec.Command("/usr/local/bin/git", "update-ref", "refs/heads/master", string(result.CommitHash)).Run()
    if err == nil {
      exec.Command("/usr/local/bin/git", "push", "origin", "master").Run()
      return true
    }
  }
  return false
}

func main() {
  err := ResetState()
  if err != nil {
    fmt.Printf("There was an issue resetting the state: %s\n", err.Error())
    return
  } else {
    fmt.Printf("State reset\n")
  }
  err = UpdateLedger()
  if err != nil {
    fmt.Printf("There was an issue updating the ledger: %s\n", err.Error())
    return
  } else {
    fmt.Printf("Ledger updated\n")
  }
  difficulty := GetDifficulty()
  tree := GetTree()
  parent := GetParent()
  t := GetTime()
  body := PrebuildBody(tree, parent, t)

  gothreadCount := 8
  result_stream := make(chan Result, gothreadCount*100)
  for i := 0; i < gothreadCount; i++ {
    go Solver(difficulty, body, result_stream)
    fmt.Printf("Launched Solver #%d\n", i)
  }

  var res Result
  for ;; {
    res = <- result_stream
    if res.Success {
      if GetGitCoin(res) {
        fmt.Printf("Success! %s\n", res.CommitHash)
      } else {
        fmt.Printf("Had some issue adding hash :( %s", res.CommitHash)
      }
      return
    }
  }
}