package codebuild

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/codebuild"
	"github.com/remind101/conveyor/builder"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/net/context"
)

var ctx = context.Background()

func TestBuilder_Build(t *testing.T) {
	c := new(mockCodeBuild)
	b := &Builder{
		BuildSpec:   template.Must(template.New("buildspec.yml").Parse(DefaultBuildspec)),
		ProjectName: ProjectName,
		codebuild:   c,
		logReader: func(group, stream string) (io.ReadCloser, error) {
			return ioutil.NopCloser(strings.NewReader(logs)), nil
		},
	}

	c.On("StartBuild", &codebuild.StartBuildInput{
		ProjectName:   aws.String("remind101-acme-inc-6Xhbd3oyMlG"),
		SourceVersion: aws.String("6af239b55ee2cfb388085d3797129c4ed88d2f5a"),
	}).Return(&codebuild.StartBuildOutput{
		Build: &codebuild.Build{
			Id: aws.String("0f19962d-7300-461b-b3da-49bad396f34f"),
		},
	}, nil).Once()

	c.On("BatchGetBuilds", &codebuild.BatchGetBuildsInput{
		Ids: []*string{aws.String("0f19962d-7300-461b-b3da-49bad396f34f")},
	}).Return(&codebuild.BatchGetBuildsOutput{
		Builds: []*codebuild.Build{
			&codebuild.Build{
				Id: aws.String("0f19962d-7300-461b-b3da-49bad396f34f"),
				Logs: &codebuild.LogsLocation{
					GroupName:  aws.String("/codebuild/conveyor-remind101-acme-inc"),
					StreamName: aws.String("abcd"),
				},
			},
		},
	}, nil).Once()

	c.On("BatchGetBuilds", &codebuild.BatchGetBuildsInput{
		Ids: []*string{aws.String("0f19962d-7300-461b-b3da-49bad396f34f")},
	}).Return(&codebuild.BatchGetBuildsOutput{
		Builds: []*codebuild.Build{
			&codebuild.Build{
				Id:            aws.String("0f19962d-7300-461b-b3da-49bad396f34f"),
				BuildComplete: aws.Bool(true),
				BuildStatus:   aws.String(codebuild.StatusTypeSucceeded),
				Logs: &codebuild.LogsLocation{
					GroupName:  aws.String("/codebuild/conveyor-remind101-acme-inc"),
					StreamName: aws.String("abcd"),
				},
			},
		},
	}, nil).Once()

	buf := new(bytes.Buffer)
	_, err := b.Build(ctx, buf, builder.BuildOptions{
		Repository: "remind101/acme-inc",
		Sha:        "6af239b55ee2cfb388085d3797129c4ed88d2f5a",
		Branch:     "master",
	})
	assert.NoError(t, err)

	assert.Equal(t, `conveyor: Waiting for CodeBuild build 0f19962d-7300-461b-b3da-49bad396f34f to start
conveyor: CodeBuild build 0f19962d-7300-461b-b3da-49bad396f34f started, streaming logs...
`+logs+`conveyor: CodeBuild build 0f19962d-7300-461b-b3da-49bad396f34f completed
`, buf.String())

	c.AssertExpectations(t)
}

func TestBuilder_Build_Fail(t *testing.T) {
	c := new(mockCodeBuild)
	b := &Builder{
		BuildSpec:   template.Must(template.New("buildspec.yml").Parse(DefaultBuildspec)),
		ProjectName: ProjectName,
		codebuild:   c,
		logReader: func(group, stream string) (io.ReadCloser, error) {
			return ioutil.NopCloser(strings.NewReader(logs)), nil
		},
	}

	c.On("StartBuild", &codebuild.StartBuildInput{
		ProjectName:   aws.String("remind101-acme-inc-6Xhbd3oyMlG"),
		SourceVersion: aws.String("6af239b55ee2cfb388085d3797129c4ed88d2f5a"),
	}).Return(&codebuild.StartBuildOutput{
		Build: &codebuild.Build{
			Id: aws.String("0f19962d-7300-461b-b3da-49bad396f34f"),
		},
	}, nil).Once()

	c.On("BatchGetBuilds", &codebuild.BatchGetBuildsInput{
		Ids: []*string{aws.String("0f19962d-7300-461b-b3da-49bad396f34f")},
	}).Return(&codebuild.BatchGetBuildsOutput{
		Builds: []*codebuild.Build{
			&codebuild.Build{
				Id: aws.String("0f19962d-7300-461b-b3da-49bad396f34f"),
				Logs: &codebuild.LogsLocation{
					GroupName:  aws.String("/codebuild/conveyor-remind101-acme-inc"),
					StreamName: aws.String("abcd"),
				},
			},
		},
	}, nil).Once()

	c.On("BatchGetBuilds", &codebuild.BatchGetBuildsInput{
		Ids: []*string{aws.String("0f19962d-7300-461b-b3da-49bad396f34f")},
	}).Return(&codebuild.BatchGetBuildsOutput{
		Builds: []*codebuild.Build{
			&codebuild.Build{
				Id:            aws.String("0f19962d-7300-461b-b3da-49bad396f34f"),
				BuildComplete: aws.Bool(true),
				BuildStatus:   aws.String(codebuild.StatusTypeFailed),
				Logs: &codebuild.LogsLocation{
					GroupName:  aws.String("/codebuild/conveyor-remind101-acme-inc"),
					StreamName: aws.String("abcd"),
				},
			},
		},
	}, nil).Once()

	buf := new(bytes.Buffer)
	_, err := b.Build(ctx, buf, builder.BuildOptions{
		Repository: "remind101/acme-inc",
		Sha:        "6af239b55ee2cfb388085d3797129c4ed88d2f5a",
		Branch:     "master",
	})
	assert.Error(t, err)

	assert.Equal(t, `conveyor: Waiting for CodeBuild build 0f19962d-7300-461b-b3da-49bad396f34f to start
conveyor: CodeBuild build 0f19962d-7300-461b-b3da-49bad396f34f started, streaming logs...
`+logs, buf.String())

	c.AssertExpectations(t)
}

func TestBuilder_Build_ProjectDoesNotExist(t *testing.T) {
	c := new(mockCodeBuild)
	b := &Builder{
		BuildSpec:   template.Must(template.New("buildspec.yml").Parse(DefaultBuildspec)),
		ProjectName: ProjectName,
		codebuild:   c,
		logReader: func(group, stream string) (io.ReadCloser, error) {
			return ioutil.NopCloser(strings.NewReader(logs)), nil
		},
	}

	c.On("StartBuild", &codebuild.StartBuildInput{
		ProjectName:   aws.String("remind101-acme-inc-6Xhbd3oyMlG"),
		SourceVersion: aws.String("6af239b55ee2cfb388085d3797129c4ed88d2f5a"),
	}).Return(&codebuild.StartBuildOutput{}, awserr.New("ResourceNotFoundException", "not found", nil)).Once()

	c.On("CreateProject", &codebuild.CreateProjectInput{
		Artifacts: &codebuild.ProjectArtifacts{
			Type: aws.String("NO_ARTIFACTS"),
		},
		Environment: &codebuild.ProjectEnvironment{
			ComputeType: aws.String("BUILD_GENERAL1_SMALL"),
			Image:       aws.String("aws/codebuild/docker:1.12.1"),
			Type:        aws.String("LINUX_CONTAINER"),
		},
		Name: aws.String("remind101-acme-inc-6Xhbd3oyMlG"),
		Source: &codebuild.ProjectSource{
			Type: aws.String("GITHUB"),
			Auth: &codebuild.SourceAuth{
				Type: aws.String("OAUTH"),
			},
			Location: aws.String("https://github.com/remind101/acme-inc"),
		},
		ServiceRole: aws.String(""),
	}).Return(&codebuild.CreateProjectOutput{}, nil).Once()

	c.On("StartBuild", &codebuild.StartBuildInput{
		ProjectName:   aws.String("remind101-acme-inc-6Xhbd3oyMlG"),
		SourceVersion: aws.String("6af239b55ee2cfb388085d3797129c4ed88d2f5a"),
	}).Return(&codebuild.StartBuildOutput{
		Build: &codebuild.Build{
			Id: aws.String("0f19962d-7300-461b-b3da-49bad396f34f"),
		},
	}, nil).Once()

	c.On("BatchGetBuilds", &codebuild.BatchGetBuildsInput{
		Ids: []*string{aws.String("0f19962d-7300-461b-b3da-49bad396f34f")},
	}).Return(&codebuild.BatchGetBuildsOutput{
		Builds: []*codebuild.Build{
			&codebuild.Build{
				Id: aws.String("0f19962d-7300-461b-b3da-49bad396f34f"),
				Logs: &codebuild.LogsLocation{
					GroupName:  aws.String("/codebuild/conveyor-remind101-acme-inc"),
					StreamName: aws.String("abcd"),
				},
			},
		},
	}, nil).Once()

	c.On("BatchGetBuilds", &codebuild.BatchGetBuildsInput{
		Ids: []*string{aws.String("0f19962d-7300-461b-b3da-49bad396f34f")},
	}).Return(&codebuild.BatchGetBuildsOutput{
		Builds: []*codebuild.Build{
			&codebuild.Build{
				Id:            aws.String("0f19962d-7300-461b-b3da-49bad396f34f"),
				BuildComplete: aws.Bool(true),
				BuildStatus:   aws.String(codebuild.StatusTypeSucceeded),
				Logs: &codebuild.LogsLocation{
					GroupName:  aws.String("/codebuild/conveyor-remind101-acme-inc"),
					StreamName: aws.String("abcd"),
				},
			},
		},
	}, nil).Once()

	buf := new(bytes.Buffer)
	_, err := b.Build(ctx, buf, builder.BuildOptions{
		Repository: "remind101/acme-inc",
		Sha:        "6af239b55ee2cfb388085d3797129c4ed88d2f5a",
		Branch:     "master",
	})
	assert.NoError(t, err)

	c.AssertExpectations(t)
}

func TestBuilder_Build_Cancel(t *testing.T) {
	ctx, cancel := context.WithCancel(ctx)

	cancelled, cleanup := untilDone(ctx)
	defer cleanup()

	c := new(mockCodeBuild)
	b := &Builder{
		BuildSpec:   template.Must(template.New("buildspec.yml").Parse(DefaultBuildspec)),
		ProjectName: ProjectName,
		codebuild:   c,
		logReader: func(group, stream string) (io.ReadCloser, error) {
			// Cancel the context when we go to read the logs.
			cancel()
			return ioutil.NopCloser(strings.NewReader(logs)), nil
		},
	}

	c.On("StartBuild", &codebuild.StartBuildInput{
		ProjectName:   aws.String("remind101-acme-inc-6Xhbd3oyMlG"),
		SourceVersion: aws.String("6af239b55ee2cfb388085d3797129c4ed88d2f5a"),
	}).Return(&codebuild.StartBuildOutput{
		Build: &codebuild.Build{
			Id: aws.String("0f19962d-7300-461b-b3da-49bad396f34f"),
		},
	}, nil).Once()

	c.On("BatchGetBuilds", &codebuild.BatchGetBuildsInput{
		Ids: []*string{aws.String("0f19962d-7300-461b-b3da-49bad396f34f")},
	}).Return(&codebuild.BatchGetBuildsOutput{
		Builds: []*codebuild.Build{
			&codebuild.Build{
				Id: aws.String("0f19962d-7300-461b-b3da-49bad396f34f"),
				Logs: &codebuild.LogsLocation{
					GroupName:  aws.String("/codebuild/conveyor-remind101-acme-inc"),
					StreamName: aws.String("abcd"),
				},
			},
		},
	}, nil).Once()

	c.On("StopBuild", &codebuild.StopBuildInput{
		Id: aws.String("0f19962d-7300-461b-b3da-49bad396f34f"),
	}).Return(&codebuild.StopBuildOutput{}, nil).Once()

	c.On("BatchGetBuilds", &codebuild.BatchGetBuildsInput{
		Ids: []*string{aws.String("0f19962d-7300-461b-b3da-49bad396f34f")},
	}).Return(&codebuild.BatchGetBuildsOutput{
		Builds: []*codebuild.Build{
			&codebuild.Build{
				Id:            aws.String("0f19962d-7300-461b-b3da-49bad396f34f"),
				BuildComplete: aws.Bool(true),
				BuildStatus:   aws.String(codebuild.StatusTypeStopped),
				Logs: &codebuild.LogsLocation{
					GroupName:  aws.String("/codebuild/conveyor-remind101-acme-inc"),
					StreamName: aws.String("abcd"),
				},
			},
		},
	}, nil).Once().WaitUntil(cancelled)

	buf := new(bytes.Buffer)
	_, err := b.Build(ctx, buf, builder.BuildOptions{
		Repository: "remind101/acme-inc",
		Sha:        "6af239b55ee2cfb388085d3797129c4ed88d2f5a",
		Branch:     "master",
	})
	assert.Equal(t, &builder.BuildCanceledError{
		Err:    context.Canceled,
		Reason: context.Canceled,
	}, err)

	c.AssertExpectations(t)
}

func TestCleanProjectName(t *testing.T) {
	tests := []struct {
		in, out string
	}{
		{"acme-inc", "acme-inc"},
		{"remind101/acme-inc", "remind101-acme-inc"},
		{"remind101/acme-inc", "remind101-acme-inc"},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			out := CleanProjectName(tt.in)
			assert.Equal(t, tt.out, out)
		})
	}
}

func TestDefaultBuildspec(t *testing.T) {
	tmpl := template.Must(template.New("buildspec.yml").Parse(DefaultBuildspec))

	tests := []struct {
		opts BuildOptions
		out  string
	}{
		{
			BuildOptions{
				BuildOptions: builder.BuildOptions{
					Repository: "remind101/acme-inc",
					Sha:        "6af239b55ee2cfb388085d3797129c4ed88d2f5a",
					Branch:     "master",
				},
				DockerCfg: "conveyor.dockercfg",
			},
			`version: 0.1
phases:
  pre_build:
    commands:

      - aws ssm get-parameters --name conveyor.dockercfg --with-decryption --query Parameters[0].Value --output text > $HOME/.dockercfg


      - docker pull "remind101/acme-inc:master" || docker pull "remind101/acme-inc:master" || docker pull "remind101/acme-inc:latest" || true

  build:
    commands:
      - docker build -t "remind101/acme-inc" .
      - docker tag "remind101/acme-inc" "remind101/acme-inc:6af239b55ee2cfb388085d3797129c4ed88d2f5a"

      - docker tag "remind101/acme-inc" "remind101/acme-inc:master"

  post_build:
    commands:

      - docker push "remind101/acme-inc:6af239b55ee2cfb388085d3797129c4ed88d2f5a"
      - docker push "remind101/acme-inc:latest"

      - docker push "remind101/acme-inc:master"

      - echo "Done pushing to docker registry"

`,
		},
		{
			BuildOptions{
				BuildOptions: builder.BuildOptions{
					Repository: "remind101/acme-inc",
					Sha:        "6af239b55ee2cfb388085d3797129c4ed88d2f5a",
				},
				DryRun: true,
			},
			`version: 0.1
phases:
  pre_build:
    commands:


      - docker pull "remind101/acme-inc:master" || docker pull "remind101/acme-inc:latest" || true

  build:
    commands:
      - docker build -t "remind101/acme-inc" .
      - docker tag "remind101/acme-inc" "remind101/acme-inc:6af239b55ee2cfb388085d3797129c4ed88d2f5a"

  post_build:
    commands:

      - echo "Dry run enabled. Not pushing"

`,
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			buf := new(bytes.Buffer)
			err := tmpl.Execute(buf, tt.opts)
			assert.NoError(t, err)
			if !assert.Equal(t, tt.out, buf.String()) && testing.Verbose() {
				io.Copy(os.Stdout, buf)
			}
		})
	}
}

type mockCodeBuild struct {
	mock.Mock
}

func (m *mockCodeBuild) StartBuild(input *codebuild.StartBuildInput) (*codebuild.StartBuildOutput, error) {
	// Don't check this for now
	input.BuildspecOverride = nil
	args := m.Called(input)
	return args.Get(0).(*codebuild.StartBuildOutput), args.Error(1)
}

func (m *mockCodeBuild) StopBuild(input *codebuild.StopBuildInput) (*codebuild.StopBuildOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*codebuild.StopBuildOutput), args.Error(1)
}

func (m *mockCodeBuild) CreateProject(input *codebuild.CreateProjectInput) (*codebuild.CreateProjectOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*codebuild.CreateProjectOutput), args.Error(1)
}

func (m *mockCodeBuild) BatchGetBuilds(input *codebuild.BatchGetBuildsInput) (*codebuild.BatchGetBuildsOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*codebuild.BatchGetBuildsOutput), args.Error(1)
}

func untilDone(ctx context.Context) (chan time.Time, func()) {
	cancelled := make(chan time.Time)
	cancel := func() { close(cancelled) }
	go func() {
		select {
		case <-ctx.Done():
			cancelled <- time.Now()
		case <-cancelled:
		}
	}()
	return cancelled, cancel
}

const logs = `[Container] 2017/05/11 02:56:56 Phase is DOWNLOAD_SOURCE                                                                                      [109/1949]
[Container] 2017/05/11 02:56:56 Source is located at /tmp/src275650405/src
[Container] 2017/05/11 02:56:56 YAML location is /codebuild/readonly/buildspec.yml
[Container] 2017/05/11 02:56:56 Registering with agent
[Container] 2017/05/11 02:56:56 Phases found in YAML: 3
[Container] 2017/05/11 02:56:56  PRE_BUILD: 1 commands
[Container] 2017/05/11 02:56:56  BUILD: 1 commands
[Container] 2017/05/11 02:56:56  POST_BUILD: 1 commands
[Container] 2017/05/11 02:56:56 Phase complete: DOWNLOAD_SOURCE Success: true
[Container] 2017/05/11 02:56:56 Phase context status code:  Message:
[Container] 2017/05/11 02:56:56 Processing plaintext environment variables
[Container] 2017/05/11 02:56:56 Processing build-level environment variables
[Container] 2017/05/11 02:56:56 {}
[Container] 2017/05/11 02:56:56 Processing builtin environment variables
[Container] 2017/05/11 02:56:56 Moving to directory /tmp/src275650405/src
[Container] 2017/05/11 02:56:56 Entering phase PRE_BUILD
[Container] 2017/05/11 02:56:56 Running command echo "Log into Docker"
Log into Docker

[Container] 2017/05/11 02:56:56 Phase complete: PRE_BUILD Success: true
[Container] 2017/05/11 02:56:56 Phase context status code:  Message:
[Container] 2017/05/11 02:56:56 Entering phase BUILD
[Container] 2017/05/11 02:56:56 Running command echo "Perform build"
Perform build

[Container] 2017/05/11 02:56:56 Phase complete: BUILD Success: true
[Container] 2017/05/11 02:56:56 Phase context status code:  Message:
[Container] 2017/05/11 02:56:56 Entering phase POST_BUILD
[Container] 2017/05/11 02:56:56 Running command echo "Push"
Push

[Container] 2017/05/11 02:56:56 Phase complete: POST_BUILD Success: true
[Container] 2017/05/11 02:56:56 Phase context status code:  Message:
[Container] 2017/05/11 02:56:57 Preparing to copy artifacts
[Container] 2017/05/11 02:56:57 No artifact files specified
`
