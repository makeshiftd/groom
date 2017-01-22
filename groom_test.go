package main

import (
    "io"
    "os"
    "flag"
    "bytes"
    "testing"
    "os/exec"
)

func TestMain(m *testing.M) {
    if os.Getenv("EXEC_GROOM") == "TRUE" {
        os.Exit(groom(os.Args[1:]))
    }
    flag.Parse()
	os.Exit(m.Run())
}


func GroomCmd(args ...string) *exec.Cmd {
    cmd := exec.Command(os.Args[0], args...)
    cmd.Env = []string{ "EXEC_GROOM=TRUE"}
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


func TestArgTmpl1(t *testing.T) {
    output, err := GroomCmd("--greeting=Hello World", "test/tmpl1.grm").Output()
    if err != nil {
        switch err := err.(type) {
        case *exec.ExitError:
            t.Fatal("Groom exited with error:\n", string(err.Stderr))
        default:
            t.Fatal("Error running groom command:\n", err)
        }
    }

    result := []byte(`<html>
    <body>Hello World</body>
</html>
`)

    if bytes.Compare(output, result) != 0 {
        t.Fatal("Output does not match expected result:\n", string(result))
    }
}


func TestStdinTmpl1(t *testing.T) {

    tmpl, oerr := os.Open("test/tmpl1.grm")
    if oerr != nil {
        t.Fatal("Error reading template:", oerr)
    }
    defer tmpl.Close()

    cmd, cerr := GroomStdin(tmpl, "--greeting=Hello World")
    if cerr != nil {
        t.Fatal("Error creating cmd:", cerr)
    }

    output, err := cmd.Output()
    if err != nil {
        switch err := err.(type) {
        case *exec.ExitError:
            t.Fatal("Groom exited with error:\n", string(err.Stderr))
        default:
            t.Fatal("Error running groom command:", err)
        }
    }

    result := []byte(`<html>
    <body>Hello World</body>
</html>
`)

    if bytes.Compare(output, result) != 0 {
        t.Fatal("Output does not match expected result:\n", string(result))
    }
}
