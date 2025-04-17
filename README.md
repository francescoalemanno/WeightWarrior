
# 🥷 WeightWarrior

_A no-nonsense TDEE tracker and weight trend analyzer for quantified self enthusiasts._

**WeightWarrior** estimates your Total Daily Energy Expenditure (TDEE) and tracks your weight trend using lightweight statistical filtering. It takes a simple CSV-like log of your weight and calorie intake, and gives back meaningful insights like trend weight, weekly change rate, and energy balance.

> Think of it as a minimalistic, terminal-native MacroFactor, powered by Go.

---

## ✨ Features

- 📉 **Two algorithms** for estimating TDEE:
  - `V1`: classic exponential smoothing-based estimation
  - `V2` (default): least-error smoothing with trend prediction
- 📊 **Daily output** with trend weight, weekly rate of change, and estimated TDEE
- 🧠 **7700 kcal/kg energy model** for body mass changes
- 🗂️ Robust to missing or incomplete data

---

## 📦 Installation

```bash
go install github.com/francescoalemanno/WeightWarrior@latest
```

Make sure `$GOPATH/bin` is in your `PATH`.

---

## 📋 Input Format

Input is a plain text file with three columns, **whitespace-separated**:

```txt
YYYY-MM-DD  WEIGHT(KG)  CALORIES(KCAL)
2025-04-10  72.4        2350
2025-04-11  72.6        2400
2025-04-12  72.5        2200
```

⚠️ **Note**: at least 3 days of data are required to produce output.

---

## 🚀 Usage

```bash
WeightWarrior [flags] <logfile>
```

### Flags

- `-a V1` – use algorithm V1 (default is V2)

### Example

```bash
WeightWarrior my_log.txt
```

Sample output:

```txt
2025-04-10 72.4 2350 - TDEE = 2350 - Trend weight: 72.4  - change per week: 0.00
2025-04-11 72.6 2400 - TDEE = 2337 - Trend weight: 72.44 - change per week: 0.01 
2025-04-12 72.5 2200 - TDEE = 2329 - Trend weight: 72.45 - change per week: 0.02 
```

---

## ⚙️ How it works

WeightWarrior fits smooth curves to both your weight and calorie data using a **recursive linear exponential smoother (LES)**. It estimates your weight trend and infers caloric surplus or deficit using the well-known `7700 kcal/kg` relationship.

- **V1**: EMA-based smoothing and online regression
- **V2**: Golden-section search to fit optimal LES smoothing factors for weight & calorie curves

---

## 📈 What's next?

- Export JSON/CSV for plotting
- Tweakable priors and smoothing constants
- CLI graph output (e.g. using `termui` or `asciigraph`)
- REST API?

---

## 🧠 Philosophy

> *"What gets measured, gets managed."*  
WeightWarrior is for those who want precise, interpretable feedback from their health data — without cloud syncing, subscriptions, or distractions.

---

## 📝 License

GPL © [Francesco Alemanno](https://github.com/francescoalemanno)
