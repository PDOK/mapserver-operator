# Local third-party replacements

`traefik-dynamic-ext` is a local compatibility module for Traefik's unpublished `github.com/traefik/traefik/dynamic/ext` module. It provides the `HTTP` and `Router` marker types imported by Traefik v3.7.x. The root `go.mod` replaces the placeholder dependency with this module.

Keep the shim aligned with Traefik's extension types and remove it when Traefik publishes the module for downstream use.