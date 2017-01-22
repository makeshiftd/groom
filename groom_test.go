package main

import (
    "io"
    "os"
    "flag"
    "bytes"
    "testing"
    "os/exec"
)


var result = []byte(`<html>
    <body>Hello World</body>
</html>
`)


func TestMain(m *testing.M) {
    if os.Getenv("EXEC_GROOM") == "TRUE" {
        os.Exit(groom(os.Args[1:]))
    }
    flag.Parse()
	os.Exit(m.Run())
}


func GroomCmd(args ...string) *exec.Cmd {
    cmd := exec.Command(os.Args[0], args...)
    cmd.Env = append(os.Environ(), "EXEC_GROOM=TRUE")
    return cmd
}

func GroomStdin(r io.Reader, args ...string) (*exec.Cmd, error) {
    cmd := GroomCmd(args...)
    stdin, err := cmd.StdinPipe()
    if err != nil {
        return nil, err
    }
    go func() {
        io.Copy(stdin, r)
        stdin.Close()
    }()
    return cmd, nil
}

func CompareOutput(t *testing.T, cmd *exec.Cmd, data []byte) {
    output, err := cmd.Output()
    if err != nil {
        switch err := err.(type) {
        default:
            t.Fatal("Command failed to start:\n", err)
        case *exec.ExitError:
            t.Fatal("Command failed with error:\n", string(err.Stderr))
        }
    }
    if bytes.Compare(output, result) != 0 {
        t.Fatal("Command output comparison failed:\n", string(result), "\nExpecting:\n", string(data))
    }
    return
}


func TestArgTmpl1(t *testing.T) {
    cmd := GroomCmd("--greeting=Hello World", "test/tmpl1.grm")

    CompareOutput(t, cmd, result)
}

func TestStdinTmpl1(t *testing.T) {
    tmpl, oerr := os.Open("test/tmpl1.grm")
    if oerr != nil {
        t.Fatal("Error reading template:", oerr)
    }
    defer tmpl.Close()

    cmd, cerr := GroomStdin(tmpl, "--greeting=Hello World")
    if cerr != nil {
        t.Fatal("Error creating command:", cerr)
    }

    CompareOutput(t, cmd, result)
}

func TestExecFunc1(t *testing.T) {
    cmd := GroomCmd("--greeting=Hello World", "test/exec1.grm")

    CompareOutput(t, cmd, result)
}

func TestStdinFunc1(t *testing.T) {
    data := bytes.NewBuffer([]byte("Hello World"))

    cmd, cerr := GroomStdin(data, "test/stdin1.grm")
    if cerr != nil {
        t.Fatal("Error creating command:", cerr)
    }

    CompareOutput(t, cmd, result)
}

func TestJsonFunc1(t *testing.T) {
    data := bytes.NewBuffer([]byte(`{ "greeting":"Hello World" }`))

    cmd, cerr := GroomStdin(data, "test/json1.grm")
    if cerr != nil {
        t.Fatal("Error creating command:", cerr)
    }

    CompareOutput(t, cmd, result)
}

func TestCatFunc1(t *testing.T) {
    cmd := GroomCmd("--data=test/cat1.json", "test/cat1.grm")

    CompareOutput(t, cmd, result)
}
