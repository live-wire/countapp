// Persisting Go objects to disc
package utils


import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "os"
    "sync"
)

var lock sync.Mutex

// Marshal is a function that marshals the object into a bytes representation
var Encode = func(v interface{}) (io.Reader, error) {
  b, err := json.MarshalIndent(v, "", "\t")
  if err != nil {
    return nil, err
  }
  return bytes.NewReader(b), nil
}

// Unmarshal is a function that unmarshals the data from the bytes representation
var Decode = func(r io.Reader, v interface{}) error {
  return json.NewDecoder(r).Decode(v)
}

// Load loads the file at path into v.
func Load(path string, v interface{}) error {
  lock.Lock()
  defer lock.Unlock()
  f, err := os.Open(path)
  if err != nil {
    fmt.Println(err)
    return err
  }
  defer f.Close()
  return Decode(f, v)
}

// Save saves a representation of v to the file at path.
func Save(path string, v interface{}) error {
  lock.Lock()
  defer lock.Unlock()
  f, err := os.Create(path)
  if err != nil {
    return err
  }
  defer f.Close()
  r, err := Encode(v)
  if err != nil {
    return err
  }
  _, err = io.Copy(f, r)
  return err
}

// Loads a path, runs the merge function and saves the resulting JSON
func LoadMergeSaveAtomic(path string, v *map[string]bool, fn func(*map[string]bool)) error {
  lock.Lock()
  defer lock.Unlock()
  f, err := os.Open(path)
  if err != nil {
    fmt.Println(err)
    return err
  }
  Decode(f, v)
  f.Close()
  fn(v)
  f2, err := os.Create(path)
  if err != nil {
    return err
  }
  defer f2.Close()
  r, err := Encode(v)
  if err != nil {
    return err
  }
  _, err = io.Copy(f2, r)
  return err
}

