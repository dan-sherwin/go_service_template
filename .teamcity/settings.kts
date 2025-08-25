import jetbrains.buildServer.configs.kotlin.v2019_2.*
import jetbrains.buildServer.configs.kotlin.v2019_2.buildSteps.script
import jetbrains.buildServer.configs.kotlin.v2019_2.triggers.vcs

version = "2025.07"


project {
    description = "CI for go_service_template"

    buildType(Build)
    buildType(Deploy)
}

object Build : BuildType({
    name = "Build & Test (Linux/amd64)"

    params {
        // set by TeamCity or override
        param("env.GOFLAGS", "-mod=mod")
        param("app.name", "service_template")
        param("build.version", "%build.number%")
    }

    artifactRules = "dist => dist"

    vcs {
        root(DslContext.settingsRoot)
        cleanCheckout = true
    }

    steps {
        script {
            name = "Setup Go"
            scriptContent = """
                set -e
                go version
                go env
            """.trimIndent()
        }
        script {
            name = "Dependencies"
            scriptContent = """
                set -e
                go mod tidy
                go vet ./...
            """.trimIndent()
        }
        script {
            name = "Tests"
            scriptContent = """
                set -e
                go test -race -count=1 ./...
            """.trimIndent()
        }
        script {
            name = "Build linux/amd64"
            scriptContent = """
                set -e
                mkdir -p dist
                export GOOS=linux
                export GOARCH=amd64
                export CGO_ENABLED=1
                LDFLAGS="-X 'scm.dev.dsherwin.net/dsherwin/go_service_template/cmd/app/consts.Version=%build.version%' \
                         -X 'scm.dev.dsherwin.net/dsherwin/go_service_template/cmd/app/consts.Commit=%build.vcs.number%' \
                         -X 'scm.dev.dsherwin.net/dsherwin/go_service_template/cmd/app/consts.BuildDate=$(date)'"
                go build -ldflags "${'$'}LDFLAGS" -o dist/%app.name% ./cmd
            """.trimIndent()
        }
        script {
            name = "Docker Build (optional)"
            enabled = false
            scriptContent = """
                set -e
                docker build -t %app.name%:%build.version% .
            """.trimIndent()
        }
    }

    requirements {
        contains("teamcity.agent.jvm.os.name", "Linux")
    }
})

object Deploy : BuildType({
    //TODO Change deployment host
    name = "Deploy to CHANGEME"
    description = "Deploy built binary via SCP/rsync and restart systemd service."

    params {
        // Customize these for your environment
        param("app.name", "service_template")
        param("deploy.dest_user", "dsherwin")
        //TODO Change deployment host
        param("deploy.dest_host", "")
        // Path to the destination binary file on the remote host
        param("deploy.dest_path", "/usr/local/%app.name%/%app.name%")
        // Service name to restart; defaults to app name
        param("service.name", "%app.name%")
    }

    vcs {
        root(DslContext.settingsRoot)
        checkoutMode = CheckoutMode.ON_SERVER
    }

    dependencies {
        artifacts(Build) {
            cleanDestination = true
            // Fetch the compiled binary from Build artifacts and place it as a local file named %app.name%
            artifactRules = "dist/%app.name%"
        }
        snapshot(Build) {}
    }

    steps {
        script {
            name = "RSYNC deploy"
            scriptContent = """
                set -euo pipefail
                BINARY="%app.name%"
                DEST_USER="%deploy.dest_user%"
                DEST_HOST="%deploy.dest_host%"
                DEST_PATH="%deploy.dest_path%"
                SERVICE_NAME="%service.name%"
                if [ ! -f "${'$'}{BINARY}" ]; then
                  echo "Binary not found at ${'$'}{BINARY}" >&2
                  exit 1
                fi
                echo "Deploying ${'$'}{BINARY} to ${'$'}{DEST_USER}@${'$'}{DEST_HOST}:${'$'}{DEST_PATH}"
                rsync -avh --stats --chmod=755 "${'$'}{BINARY}" "${'$'}{DEST_USER}@${'$'}{DEST_HOST}:${'$'}{DEST_PATH}"
                echo "Restarting service ${'$'}{SERVICE_NAME} on ${'$'}{DEST_HOST}"
                ssh -o BatchMode=yes "${'$'}{DEST_USER}@${'$'}{DEST_HOST}" "sudo systemctl restart ${'$'}{SERVICE_NAME} || systemctl restart ${'$'}{SERVICE_NAME}"
            """.trimIndent()
        }
    }

    requirements {
        // Assumes SSH keys are available for the agent's user
    }
})
