{
  description = "A blazingly fast Slack TUI";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs {
          inherit system;
        };
        lib = pkgs.lib;
        slk = pkgs.buildGo126Module {
          pname = "slk";
          version = "0.0.0";
          src = ./.;
          vendorHash = "sha256-/J4gr4m9v6Y0Be8BU4wepIdl2sjoPh0pFCvJL2kIeLk=";
          buildInputs = [pkgs.libX11];
          # The test suite spins up httptest servers on loopback. The macOS
          # Nix sandbox denies all networking by default, so allow loopback
          # there; the attribute is a no-op on Linux.
          __darwinAllowLocalNetworking = true;
        };
        # VHS with sixel capture (charmbracelet/vhs#783), so the demo GIFs
        # show real images instead of half-blocks. Switch back to pkgs.vhs
        # once that PR is in a release. Go 1.26 because the fork needs Go
        # 1.25.12+, newer than this nixpkgs' go_1_25. Its tests need ttyd
        # and a browser, which the build sandbox lacks.
        vhs-sixel = (pkgs.vhs.override { buildGoModule = pkgs.buildGo126Module; }).overrideAttrs (old: {
          version = "${old.version}-sixel";
          src = pkgs.fetchFromGitHub {
            owner = "jchook";
            repo = "vhs";
            rev = "113917ffe5585f19b58596f01b141eb1ebc01b54";
            hash = "sha256-lL5+VOEGP3yiDhUO7CH2CI2Jvy09Zt12YJTiQJesSJ4=";
          };
          vendorHash = "sha256-0/GA+AGyHiw7PvUXRBuL9yylE3NOY3vVUA3uSFAP11Q=";
          doCheck = false;
        });
      in {
        packages.default = slk;
        packages.slk = slk;
        # `nix develop`: slk's build inputs (Go, libX11) plus vhs, for `make demo-gifs`.
        devShells.default = pkgs.mkShell {
          inputsFrom = [ slk ];
          packages = [ vhs-sixel ];
        };
      });
}
