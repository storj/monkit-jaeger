{
  description = "Gio build environment";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs";
  };

  outputs = { self, nixpkgs }:
    let
      supportedSystems = [ "x86_64-linux" ];
      forAllSystems = f: nixpkgs.lib.genAttrs supportedSystems (system: f system);
    in
      {
        devShells = forAllSystems (system:
            let
              pkgs = import nixpkgs { inherit system; };
            in {
              default = with pkgs; mkShell ({
                  LD_LIBRARY_PATH = "${vulkan-loader}/lib";

                  packages = [
                    vulkan-headers
                    libxkbcommon
                    wayland
                    xorg.libX11
                    xorg.libXcursor
                    xorg.libXfixes
                    libGL
                    pkgconfig
                  ];
              });
            }
          );
      };
}
