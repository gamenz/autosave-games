package steam

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

var accts = []string{"1440787955", "1518622314"}

const (
	testSteamRoot  = `D:\Steam`
	testInstallDir = `D:\Steam\steamapps\common\X`
)

func TestSaveEntries(t *testing.T) {
	cases := []struct {
		name  string
		files []CloudFile
		want  []string
		warn  int
	}{
		{"elden ring two accounts", []CloudFile{
			{"1440787955", 4, "EldenRing/76561199478888042/ER0000.sl2"},
			{"1518622314", 4, "EldenRing/76561198000000001/ER0000.sl2"}}, []string{"%APPDATA%/EldenRing"}, 0},
		{"unreal savegames", []CloudFile{
			{"1440787955", 3, "Sandfall/Saved/SaveGames/76561198095236296/a.sav"},
			{"1440787955", 3, "Sandfall/Saved/SaveGames/76561198095236296/b.sav"}},
			[]string{"%LOCALAPPDATA%/Sandfall/Saved/SaveGames"}, 0},
		{"account id mid path", []CloudFile{
			{"1518622314", 3, "KoeiTecmo/FatalFrameII/Savedata/1518622314/SLOT00/SAVEDATA.BIN"}},
			[]string{"%LOCALAPPDATA%/KoeiTecmo/FatalFrameII/Savedata"}, 0},
		{"files at root base", []CloudFile{{"1440787955", 4, "save.dat"}, {"1440787955", 4, "opts.ini"}},
			[]string{"%APPDATA%/opts.ini", "%APPDATA%/save.dat"}, 0},
		{"documents", []CloudFile{{"1440787955", 2, "My Games/X/s.sav"}},
			[]string{"%USERPROFILE%/Documents/My Games/X"}, 0},
		{"steam remote", []CloudFile{{"1440787955", 0, "save1.sav"}, {"1518622314", 0, "save2.sav"}},
			[]string{`D:\Steam\userdata\*\391220\remote`}, 0},
		{"install dir", []CloudFile{{"1440787955", 1, "hl2/save/quick.sav"}},
			[]string{testInstallDir + "/hl2/save"}, 0},
		{"unknown root", []CloudFile{{"1440787955", 7, "x/y.sav"}}, nil, 1},
		{"messy keys", []CloudFile{
			{"1440787955", 3, `Kena\Saved\SaveGames\a.sav`}, {"1440787955", 3, "./Kena/Saved/SaveGames/b.sav"}},
			[]string{"%LOCALAPPDATA%/Kena/Saved/SaveGames"}, 0},
		{"two roots ordered", []CloudFile{
			{"1440787955", 4, "B/x.sav"}, {"1440787955", 3, "A/y.sav"}},
			[]string{"%LOCALAPPDATA%/A", "%APPDATA%/B"}, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, warnings := SaveEntries(testSteamRoot, testInstallDir, 391220, tc.files, accts)
			if !slices.Equal(got, tc.want) || len(warnings) != tc.warn {
				t.Fatalf("SaveEntries = %v (warnings %v), want %v (%d warnings)", got, warnings, tc.want, tc.warn)
			}
		})
	}
}

func TestCloudFilesAndAccounts(t *testing.T) {
	root := t.TempDir()
	for _, a := range []string{"222", "111"} {
		writeFile(t, filepath.Join(root, "userdata", a, "1245620", "remotecache.vdf"), fmt.Sprintf(
			"\"1245620\"\n{\n\t\"ChangeNumber\"\t\"0\"\n\t\"EldenRing/7656119947888%s/ER0000.sl2\"\n\t{\n\t\t\"root\"\t\"4\"\n\t\t\"size\"\t\"1\"\n\t}\n}\n", a[:3]))
	}
	writeFile(t, filepath.Join(root, "userdata", "333", "7", "remotecache.vdf"), `"7" { }`)
	files, errs := CloudFiles(root, 1245620)
	if len(errs) != 0 || len(files) != 2 {
		t.Fatalf("CloudFiles = %+v, %v", files, errs)
	}
	for _, f := range files {
		if f.Root != 4 || !strings.HasPrefix(f.Path, "EldenRing/7656119947888") {
			t.Errorf("file = %+v", f)
		}
	}
	if got := Accounts(root); !slices.Equal(got, []string{"111", "222", "333"}) {
		t.Errorf("Accounts = %v", got)
	}
}
