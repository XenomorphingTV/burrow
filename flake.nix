{
  description = "A personal task catalogue, runner, and scheduler with a terminal UI";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs =
    { self, nixpkgs }:
    let
      supportedSystems = [
        "x86_64-linux"
        "aarch64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];

      forAllSystems = nixpkgs.lib.genAttrs supportedSystems;
    in
    {
      packages = forAllSystems (
        system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
        in
        {
          default = pkgs.buildGoModule {
            pname = "burrow";
            version = "0.1.5";

            src = ./.;

            # NOTE: Hash needs to be regularly updated with versions
            vendorHash = "sha256-+XR56yrvaqw04MVSizXQiTtxxMgfqtKkonKUER4TbL0=";

            ldflags = [
              "-s"
              "-w"
              "-X github.com/XenomorphingTV/burrow/internal/version.Version=0.1.5"
            ];

            # Install the man page automatically alongside the binary
            postInstall = ''
              install -Dm644 burrow.1 $out/share/man/man1/burrow.1
            '';

            meta = with pkgs.lib; {
              description = "A personal task catalogue, runner, and scheduler with a terminal UI";
              homepage = "https://github.com/XenomorphingTV/burrow";
              license = licenses.mit;
              mainProgram = "burrow";
            };
          };
        }
      );

      devShells = forAllSystems (
        system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
        in
        {
          default = pkgs.mkShell {
            buildInputs = with pkgs; [
              go
              gnumake
              goreleaser
            ];
          };
        }
      );
    };
}
