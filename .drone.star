def main(ctx):
    stages = [
        linux(ctx, "amd64","1.2.0"),
        linux(ctx, "arm64","1.2.0"),
        linux(ctx, "arm","1.2.0"),
    ]

    return stages

def linux(ctx, arch, version):
    build = [
        'go build -v -ldflags "-X main.version=%s" -a -tags netgo -o release/linux/%s/docker .' % (version, arch),
    ]

    steps = [
        {
            "name": "environment",
            "image": "golang:1.16.7",
            "pull": "always",
            "environment": {
                "CGO_ENABLED": "0",
            },
            "commands": [
                "go version",
                "go env",
            ],
        },
        {
            "name": "build",
            "image": "golang:1.16.7",
            "environment": {
                "CGO_ENABLED": "0",
            },
            "commands": build,
        },
        {
            "name": "executable",
            "image": "golang:1.16.7",
            "commands": [
                "./release/linux/%s/docker --help" % (arch),
            ],
        },
    ]

    steps.append({
        "name": "docker",
        "image": "plugins/docker",
        "settings": {
            "dockerfile": "docker/Dockerfile.linux.%s" % (arch),
            "repo": "xuanloc0511/drone-plugin-docker",
            "username": {
                "from_secret": "docker_username",
            },
            "password": {
                "from_secret": "docker_password",
            },
            "tags": [
                "%s-linux-%s" % (version, arch), 
                "latest-linux-%s"% (arch),
            ],
        },
    })

    return {
        "kind": "pipeline",
        "type": "docker",
        "name": "%s-linux-%s" % (version, arch),
        "steps": steps,
        "platform": {
            "os": "linux",
            "arch": arch,
        },
        "depends_on": [],
        "trigger": {
            "ref": [
                "refs/heads/main",
                "refs/tags/**",
                "refs/pull/**",
            ],
        },
    }