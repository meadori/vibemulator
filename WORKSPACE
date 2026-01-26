load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

http_archive(
    name = "io_bazel_rules_go",
    sha256 = "0798367756b15676a88e5d07c39379207e38a221a6034f5e26343513e83955da",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/rules_go/releases/download/v0.39.1/rules_go-v0.39.1.zip",
        "https://github.com/bazelbuild/rules_go/releases/download/v0.39.1/rules_go-v0.39.1.zip",
    ],
)

http_archive(
    name = "bazel_gazelle",
    sha256 = "f4c9c6451633ad0de400a51980389a9f548545800c7104b207df34eb10825316",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/bazel-gazelle/releases/download/v0.29.0/bazel-gazelle-v0.29.0.zip",
        "https://github.com/bazelbuild/bazel-gazelle/releases/download/v0.29.0/bazel-gazelle-v0.29.0.zip",
    ],
)

load("@io_bazel_rules_go//go:deps.bzl", "go_register_toolchains", "go_rules_dependencies")
load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies")

go_rules_dependencies()

go_register_toolchains()

gazelle_dependencies()

go_repository(
    name = "com_github_hajimehoshi_ebiten_v2",
    importpath = "github.com/hajimehoshi/ebiten/v2",
    sum = "h1:0000000000000000000000000000000000000000000000000000",
    version = "v2.9.7",
)

go_repository(
    name = "com_github_meadori_vibemulator",
    importpath = "github.com/meadori/vibemulator",
    sum = "h1:0000000000000000000000000000000000000000000000000000",
    version = "v0.0.0",
)
