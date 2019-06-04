package main

//go:generate go run scripts/generate.go

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/docker/pkg/term"
	"gopkg.in/yaml.v2"
)

var (
	sourceDir  string
	binaryDir  string
	prefixDir  string
	configFile string
)

func init() {
	flag.StringVar(&sourceDir, "source-dir", ".", "source directory")
	flag.StringVar(&binaryDir, "binary-dir", "build", "binary directory")
	flag.StringVar(&prefixDir, "prefix-dir", "prefix", "prefix directory")
	flag.StringVar(&configFile, "config-file", ".floyd.yaml", "configuration file")
}

type builder struct {
	Base            string            `yaml:"base"`
	APK             []string          `yaml:"apk"`
	APT             []string          `yaml:"apt"`
	RUN             []string          `yaml:"run"`
	Model           string            `yaml:"model"`
	Configurations  []string          `yaml:"configurations,flow"`
	Steps           []string          `yaml:"steps,flow"`
	CMakeGenerator  string            `yaml:"cmake-generator"`
	CheckoutCommand string            `yaml:"checkout-command"`
	UpdateCommand   string            `yaml:"update-command"`
	CoverageCommand string            `yaml:"coverage-command"`
	MemcheckCommand string            `yaml:"memcheck-command"`
	MemcheckType    string            `yaml:"memcheck-type"`
	SubmitURL       string            `yaml:"submit-url"`
	Cache           map[string]string `yaml:"cache"`
	Env             map[string]string `yaml:"env"`
}

func (b *builder) Tag() string {
	hasher := md5.New()
	fmt.Fprintf(hasher, "%s-%v-%v-%v", b.Base, b.APK, b.APT, b.RUN)
	return hex.EncodeToString(hasher.Sum(nil))
}

func main() {
	flag.Parse()
	var err error

	sourceDir, err = filepath.Abs(sourceDir)
	if err != nil {
		log.Fatal(err)
	}

	binaryDir, err = filepath.Abs(binaryDir)
	if err != nil {
		log.Fatal(err)
	}

	prefixDir, err = filepath.Abs(prefixDir)
	if err != nil {
		log.Fatal(err)
	}

	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Fatal(err)
	}

	builders := make(map[string]builder)
	err = yaml.Unmarshal(data, &builders)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.WithVersion("1.30"), client.FromEnv)
	if err != nil {
		log.Fatal(err)
	}

	for k, v := range builders {
		cacheFile := filepath.Join(binaryDir, k+"-cache.cmake")

		os.MkdirAll(filepath.Join(binaryDir, k), os.ModePerm)
		os.MkdirAll(filepath.Join(prefixDir, k), os.ModePerm)

		if err := writeCache(cacheFile, v.Cache); err != nil {
			log.Fatal(err)
		}

		image := "builder:" + v.Tag()
		if err := buildImage(ctx, cli, image, &v); err != nil {
			log.Fatal(err)
		}

		if err := execBuild(ctx, cli, image, k, &v); err != nil {
			log.Fatal(err)
		}
	}
}

func writeCache(file string, entries map[string]string) error {
	f, err := os.Create(file)
	if err != nil {
		return err
	}
	defer f.Close()
	for k, v := range entries {
		fmt.Fprintf(f, "set(\"%s\" \"%s\" CACHE INTERNAL \"\")\n", k, v)
	}
	return nil
}

func dockerfile(b *builder) string {
	var buf strings.Builder
	fmt.Fprintln(&buf, "FROM", b.Base)
	fmt.Fprintln(&buf, "COPY . /")

	if len(b.APK) > 0 {
		fmt.Fprintln(&buf, "RUN apk --no-cache add", strings.Join(b.APK, " "))
	}

	if len(b.APT) > 0 {
		fmt.Fprintln(&buf, "RUN apt-get update",
			"&& apt-get install -y --no-install-recommends", strings.Join(b.APT, " "),
			"&& rm -rf /var/lib/apt/lists/*")
	}

	if len(b.RUN) > 0 {
		fmt.Fprintln(&buf, "RUN", strings.Join(b.RUN, " "))
	}

	return buf.String()
}

func writeTarRecord(w *tar.Writer, fn, contents string) error {
	err := w.WriteHeader(&tar.Header{
		Name:     fn,
		Mode:     0644,
		Size:     int64(len(contents)),
		Typeflag: '0',
	})
	if err != nil {
		return err
	}
	_, err = w.Write([]byte(contents))
	if err != nil {
		return err
	}
	return nil
}

func buildImage(ctx context.Context, cli *client.Client, image string, b *builder) error {
	var buf bytes.Buffer
	if err := tarRC(&buf); err != nil {
		return err
	}
	w := tar.NewWriter(&buf)
	if err := writeTarRecord(w, "Dockerfile", dockerfile(b)); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}

	resp, err := cli.ImageBuild(ctx, &buf, types.ImageBuildOptions{
		Tags:       []string{image},
		PullParent: true,
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	fd, isTerminal := term.GetFdInfo(os.Stdout)
	return jsonmessage.DisplayJSONMessagesStream(resp.Body, os.Stdout, fd, isTerminal, nil)
}

func execBuild(ctx context.Context, cli *client.Client, image, name string, b *builder) error {
	hostname, err := os.Hostname()
	if err != nil {
		return err
	}

	usr, err := user.Current()
	if err != nil {
		return err
	}

	env := []string{
		"CCACHE_DIR=/ccache",
	}

	for k, v := range b.Env {
		env = append(env, k+"="+v)
	}

	config := &container.Config{
		Image:    image,
		Hostname: hostname,
		User:     usr.Uid + ":" + usr.Gid,
		Env:      env,
		Cmd: []string{
			"ctest", "-S", "/entrypoint.cmake",
			"-DBUILD_MODEL=" + b.Model,
			"-DBUILD_CONFIGURATIONS=" + strings.Join(b.Configurations, ";"),
			"-DBUILD_STEPS=" + strings.Join(b.Steps, ";"),
			"-DCTEST_BUILD_NAME=" + name,
			"-DCMAKE_GENERATOR=" + b.CMakeGenerator,
			"-DCHECKOUT_COMMAND=" + b.CheckoutCommand,
			"-DUPDATE_COMMAND=" + b.UpdateCommand,
			"-DCOVERAGE_COMMAND=" + b.CoverageCommand,
			"-DMEMCHECK_COMMAND=" + b.MemcheckCommand,
			"-DMEMCHECK_TYPE=" + b.MemcheckType,
			"-DUSE_LAUNCHERS=ON",
			"-VV",
		},
	}

	hostConfig := &container.HostConfig{
		Mounts: []mount.Mount{
			mount.Mount{
				Type:     mount.TypeBind,
				Source:   sourceDir,
				Target:   "/source",
				ReadOnly: false, // TODO: if "update" in "steps", mount source as rw, otherwise ro
			},
			mount.Mount{
				Type:     mount.TypeBind,
				Source:   binaryDir + "/" + name,
				Target:   "/binary",
				ReadOnly: false,
			},
			mount.Mount{
				Type:     mount.TypeBind,
				Source:   prefixDir + "/" + name,
				Target:   "/prefix",
				ReadOnly: false,
			},
			mount.Mount{
				Type:     mount.TypeBind,
				Source:   binaryDir + "/" + name + "-cache.cmake",
				Target:   "/cache.cmake",
				ReadOnly: true,
			},
			mount.Mount{
				Type:     mount.TypeBind,
				Source:   os.Getenv("CCACHE_DIR"),
				Target:   "/ccache",
				ReadOnly: false,
			},
			mount.Mount{
				Type:     mount.TypeBind,
				Source:   "/etc/passwd",
				Target:   "/etc/passwd",
				ReadOnly: true,
			},
		},
	}

	resp, err := cli.ContainerCreate(ctx, config, hostConfig, nil, "")
	if err != nil {
		return err
	}

	out, err := cli.ContainerAttach(ctx, resp.ID, types.ContainerAttachOptions{
		Stream: true,
		Stdin:  false,
		Stdout: true,
		Stderr: true,
		Logs:   true,
	})
	if err != nil {
		return err
	}
	defer out.Conn.Close()

	if err = cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return err
	}

	_, err = stdcopy.StdCopy(os.Stdout, os.Stderr, out.Reader)
	return err
}
