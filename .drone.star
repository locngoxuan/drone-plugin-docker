def main(ctx):
    stages = [
        linux(ctx, "amd64","1.5.0"),
        linux(ctx, "arm64","1.5.0"),
        linux(ctx, "arm","1.5.0"),
    ]

    after = manifest(ctx, "1.5.0")

    for s in stages:
        for a in after:
            a["depends_on"].append(s["name"])

    return stages + after


def manifest(ctx, version):
    return [{
        "kind": "pipeline",
        "type": "docker",
        "name": "manifest-%s" % (version),
        "steps": [{
            "name":"manifest",
            "image":"plugins/manifest",
            "settings": {
                "target": "xuanloc0511/drone-plugin-docker:%s" % (version),
                "template": "xuanloc0511/drone-plugin-docker:%s-OS-ARCH" % (version),
                "username": {
                    "from_secret": "docker_username",
                },
                "password": {
                    "from_secret": "docker_password",
                },
                "platforms":[
                    "linux/amd64",
                    "linux/arm",
                    "linux/arm64",
                ],
            },            
        }],
        "depends_on": [],
        "trigger": {
            "ref": [
                "refs/heads/main",
                "refs/tags/**",
            ],
        },
    },{
        "kind": "pipeline",
        "type": "docker",
        "name": "manifest-latest",
        "steps": [{
            "name":"manifest",
            "image":"plugins/manifest",
            "settings": {
                "target": "xuanloc0511/drone-plugin-docker:latest",
                "template": "xuanloc0511/drone-plugin-docker:latest-OS-ARCH",
                "username": {
                    "from_secret": "docker_username",
                },
                "password": {
                    "from_secret": "docker_password",
                },
                "platforms":[
                    "linux/amd64",
                    "linux/arm",
                    "linux/arm64",
                ],
            },            
        }],
        "depends_on": [],
        "trigger": {
            "ref": [
                "refs/heads/main",
                "refs/tags/**",
            ],
        },
    }]

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