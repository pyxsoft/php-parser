// Package benchcorpus provides deterministic PHP sources for benchmarks.
//
// The synthetic files model the workloads the parser sees in practice:
// ordinary OOP code, modern PHP 8 syntax, template-style inline HTML,
// interpolation-heavy strings and obfuscated (malware-like) code with huge
// literals and long expression chains.
//
// Real-world code can be added without committing it to the repository by
// pointing PHPPARSER_BENCH_CORPUS at a directory; every *.php file under it is
// loaded as an extra corpus entry.
package benchcorpus

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io/fs"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// EnvCorpusDir names the environment variable with an optional external corpus.
const EnvCorpusDir = "PHPPARSER_BENCH_CORPUS"

type File struct {
	Name string
	Src  []byte
}

// Synthetic returns the generated corpus. The output is identical on every call.
func Synthetic() []File {
	return []File{
		{"class", genClass(40)},
		{"modern", genModern(60)},
		{"template", genTemplate(400)},
		{"strings", genStrings(600)},
		{"obfuscated", genObfuscated()},
		{"minified", genMinified(3000)},
	}
}

// External loads every *.php file under the directory named by
// PHPPARSER_BENCH_CORPUS, concatenated per file. It returns nil when the
// variable is unset.
func External(tb testing.TB) []File {
	dir := os.Getenv(EnvCorpusDir)
	if dir == "" {
		return nil
	}

	var files []File
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.EqualFold(filepath.Ext(path), ".php") {
			return err
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, path)
		files = append(files, File{Name: rel, Src: src})
		return nil
	})
	if err != nil {
		tb.Fatalf("load %s: %v", EnvCorpusDir, err)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })

	return files
}

// Size returns the total byte size of files.
func Size(files []File) int64 {
	var n int64
	for _, f := range files {
		n += int64(len(f.Src))
	}
	return n
}

func genClass(n int) []byte {
	var b bytes.Buffer
	b.WriteString("<?php\n\nnamespace App\\Service;\n\nuse App\\Model\\User;\nuse App\\Repository\\UserRepository as Repo;\n\n")
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, `/**
 * Service %[1]d handles user operations.
 */
class UserService%[1]d extends BaseService implements ServiceInterface
{
    const LIMIT = %[1]d;
    private static $instances = [];
    protected $repo;
    private $cache = array('hits' => 0, 'misses' => 0);

    public function __construct(Repo $repo, $options = null)
    {
        parent::__construct($options);
        $this->repo = $repo;
        self::$instances[] = $this;
    }

    public function find($id)
    {
        if (isset($this->cache[$id])) {
            $this->cache['hits']++;
            return $this->cache[$id];
        } elseif ($id <= 0 || !is_numeric($id)) {
            throw new \InvalidArgumentException("Invalid id: {$id}");
        }
        $user = $this->repo->findOneBy(['id' => (int) $id, 'active' => true]);
        $this->cache[$id] = $user ?: null;
        return $user;
    }

    public function filter(array $users, $min = 0)
    {
        $out = [];
        foreach ($users as $k => $u) {
            switch ($u->status) {
                case 'banned':
                    continue 2;
                default:
                    break;
            }
            for ($j = 0; $j < count($u->roles); $j++) {
                $out[$k][] = strtoupper($u->roles[$j]) . '_' . self::LIMIT;
            }
            $total = $u->score * 2 + $min - ($u->penalty / 3) %% 7;
            $out[$k]['total'] = $total > 100 ? 100 : $total;
        }
        usort($out, function ($a, $b) use ($min) {
            return $a['total'] <=> $b['total'];
        });
        return array_map(fn($x) => $x['total'] ?? 0, $out);
    }

    public static function create()
    {
        try {
            return new static(new Repo(User::class));
        } catch (\Exception $e) {
            error_log(sprintf('%%s: %%s', __METHOD__, $e->getMessage()));
        } finally {
            unset($e);
        }
        return null;
    }
}

`, i)
	}
	return b.Bytes()
}

func genModern(n int) []byte {
	var b bytes.Buffer
	b.WriteString("<?php\n\ndeclare(strict_types=1);\n\nnamespace App\\Domain;\n\n")
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, `enum Status%[1]d: string implements HasLabel
{
    case Active = 'active';
    case Inactive = 'inactive';

    public function label(): string
    {
        return match ($this) {
            self::Active => 'On',
            self::Inactive => 'Off',
        };
    }
}

#[Entity(table: 'orders_%[1]d'), Cache(ttl: 60)]
final readonly class Order%[1]d
{
    public function __construct(
        public int $id,
        private ?string $note = null,
        protected int|float $amount = 0,
        public Status%[1]d $status = Status%[1]d::Active,
    ) {}

    public function total(iterable&\Countable $items, mixed ...$extra): static|null
    {
        $sum = array_sum(array_map(static fn(Item $i): int => $i?->price ?? 0, [...$items]));
        $fmt = number_format(...);
        return $sum > 0 ? new static(id: $this->id, amount: $sum) : throw new \DomainException('empty');
    }

    public function never(): never
    {
        exit(1);
    }
}

`, i)
	}
	return b.Bytes()
}

func genTemplate(n int) []byte {
	var b bytes.Buffer
	b.WriteString("<!DOCTYPE html>\n<html><head><title><?= htmlspecialchars($title) ?></title></head>\n<body>\n<ul>\n")
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, `<?php if ($items[%[1]d]['visible']): ?>
  <li class="item item-%[1]d <?php echo $active === %[1]d ? 'active' : ''; ?>">
    <a href="<?= esc_url($items[%[1]d]['url']) ?>"><?= esc_html($items[%[1]d]['label']) ?></a>
  </li>
<?php endif; ?>
`, i)
	}
	b.WriteString("</ul>\n</body></html>\n")
	return b.Bytes()
}

func genStrings(n int) []byte {
	var b bytes.Buffer
	b.WriteString("<?php\n")
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, `$s%[1]d = "Hello {$user->name}, you have $count new ${kind} messages in {$box['inbox'][%[1]d]}\n";
$h%[1]d = <<<EOT
    Dear $user->name,
    Your order {$order->id} of {$order->items[0]->name} ships on $date.
    EOT;
$n%[1]d = <<<'EOT'
raw $text {not} interpolated
EOT;
`, i)
	}
	return b.Bytes()
}

// genObfuscated mimics typical PHP malware: large encoded payloads, string
// building via chr()/concatenation, variable functions and huge arrays.
func genObfuscated() []byte {
	rnd := rand.New(rand.NewSource(1))
	payload := make([]byte, 192*1024)
	rnd.Read(payload)

	var b bytes.Buffer
	b.WriteString("<?php\n")
	fmt.Fprintf(&b, "$_x = '%s';\n", base64.StdEncoding.EncodeToString(payload))
	b.WriteString("@eval(gzinflate(base64_decode(str_rot13(strrev($_x)))));\n")

	b.WriteString("$f = ")
	for i := 0; i < 4000; i++ {
		if i > 0 {
			b.WriteString(".")
		}
		fmt.Fprintf(&b, "chr(%d)", 32+rnd.Intn(90))
	}
	b.WriteString(";\n")

	b.WriteString("$t = array(")
	for i := 0; i < 20000; i++ {
		fmt.Fprintf(&b, "%d,", rnd.Intn(256))
	}
	b.WriteString(");\n")

	b.WriteString("$k = \"")
	for i := 0; i < 8000; i++ {
		fmt.Fprintf(&b, "\\x%02x", rnd.Intn(256))
	}
	b.WriteString("\";\n")

	for i := 0; i < 300; i++ {
		fmt.Fprintf(&b, "$GLOBALS['_%d'] = $f[%d]($t[%d] ^ ord($k[%d %% strlen($k)]), ${'_'.'G'.'E'.'T'}['p%d'] ?? null);\n", i, i%7, i, i, i)
	}
	return b.Bytes()
}

func genMinified(n int) []byte {
	var b bytes.Buffer
	b.WriteString("<?php ")
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, "function f%[1]d($a,$b=%[1]d){if($a>$b){return $a*$b+%[1]d;}else{$c=[$a,$b,'k%[1]d'=>!$a];return $c[0]??$b;}}", i)
	}
	return b.Bytes()
}
