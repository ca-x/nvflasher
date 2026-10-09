package setup

import "fmt"

type Package struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	SHA1 string `json:"sha1"`
}

type Model struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Release  string    `json:"release"`
	Board    string    `json:"board"`
	Packages []Package `json:"packages"`
}

var orinNanoPackages = []Package{
	{"Jetson_Linux_R36.5.0_aarch64.tbz2", "https://developer.nvidia.com/downloads/embedded/l4t/r36_release_v5.0/release/Jetson_Linux_r36.5.0_aarch64.tbz2", "96e691a6d2d618e22dd6cb0630ee17faaa4733e9"},
	{"Tegra_Linux_Sample-Root-Filesystem_R36.5.0_aarch64.tbz2", "https://developer.nvidia.com/downloads/embedded/l4t/r36_release_v5.0/release/Tegra_Linux_Sample-Root-Filesystem_r36.5.0_aarch64.tbz2", "7844cfc00ef92eeb85d699d17bcb787a1560d486"},
}

var models = []Model{
	{ID: "orin-nano-super-devkit", Name: "Jetson Orin Nano Super Developer Kit", Release: "36.5.0", Board: "jetson-orin-nano-devkit-super", Packages: orinNanoPackages},
	{ID: "orin-nano-8gb-devkit", Name: "Jetson Orin Nano 8GB Developer Kit", Release: "36.5.0", Board: "jetson-orin-nano-devkit", Packages: orinNanoPackages},
}

func Models() []Model { return models }

func Lookup(id string) (Model, error) {
	for _, model := range models {
		if model.ID == id {
			return model, nil
		}
	}
	return Model{}, fmt.Errorf("no verified NVIDIA download for model %q", id)
}
