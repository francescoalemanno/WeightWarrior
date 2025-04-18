package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata"
)

const CAL_FAT = 7700.0
const WUNIT = "kg"

func main() {
	location, err := time.LoadLocation("UTC")
	if err != nil {
		panic(err)
	}
	time.Local = location
	algorithm := flag.String("a", "V2", "Algorithm choice V1 or V2 (default)")
	flag.Parse()

	cfg_file := flag.Arg(0)
	bts, err := os.ReadFile(cfg_file)
	if err != nil {
		bts = []byte{}
	}
	data := string(bts)

	rows := strings.Split(data, "\n")
	slices.Sort(rows)
	first := time.Time{}
	first_done := false

	dates := []string{}
	times := []float64{}
	cals := []float64{}
	weights := []float64{}
	goal_weight := math.NaN()
	for j := range rows {
		vec := strings.Fields(rows[j])
		if len(vec) == 2 && vec[0] == "gw" {
			gw, err := strconv.ParseFloat(vec[1], 64)
			if err == nil && gw >= 0 {
				goal_weight = gw
			} else {
				fmt.Println("Failed to set goal weight to ", vec[1], ", unparsable number.")
			}
		}
		if len(vec) == 3 {
			tparsed, err := time.Parse("2006-01-02", vec[0])
			if err != nil {
				continue
			}
			w, err := strconv.ParseFloat(vec[1], 64)
			if err != nil {
				continue
			}
			cal, err := strconv.ParseFloat(vec[2], 64)
			if err != nil {
				continue
			}
			if !first_done {
				first = tparsed
				first_done = true
			}
			t := float64(tparsed.Sub(first).Hours()) / 24.0
			dates = append(dates, vec[0])
			times = append(times, t)
			cals = append(cals, cal)
			weights = append(weights, w)
		}

	}
	if len(times) < 3 {
		fmt.Println("Not enough data to estimate TDEE, fill in atleast 3 days.")
		return
	}
	if *algorithm == "V1" {
		tdee, tdee_sd := TDEE_V1(times, weights, cals, 1/10.5)
		for i := range times {
			fmt.Println(dates[i], math.Round(weights[i]*10)/10, math.Round(cals[i]), "- TDEE =", math.Round(tdee[i]), "+/-", math.Round(tdee_sd[i]))
		}
		PrintGoalWeight(goal_weight, weights[len(weights)-1], tdee[len(tdee)-1])
	}
	if *algorithm == "V2" {
		w, dwdt, tdee := TDEE_V2(times, weights, cals)
		for i := range times {
			fmt.Println(dates[i], math.Round(weights[i]*10)/10, math.Round(cals[i]), "- TDEE =", math.Round(tdee[i]), "- Trend weight:", math.Round(w[i]*100)/100, "- change per week:", math.Round(dwdt[i]*7*100)/100, "")
		}
		PrintGoalWeight(goal_weight, w[len(w)-1], tdee[len(tdee)-1])
	}
}

func PrintGoalWeight(goal_weight float64, lw float64, ltdee float64) {
	if !math.IsNaN(goal_weight) {
		delta := min(max(min(max(goal_weight-lw, -lw*0.0005), lw*0.0005)*CAL_FAT, -ltdee*0.25), ltdee*0.25)
		suggested_cals := ltdee + delta
		fmt.Println("To reach goal weight of", goal_weight, WUNIT+", suggested calories:", math.Round(suggested_cals), "cal")
		fmt.Println("At this rate, you should expect a weekly change of", math.Round((suggested_cals-ltdee)/CAL_FAT*100*7)/100, WUNIT)
	}
}

func TDEE_V2(times []float64, weights []float64, cals []float64) ([]float64, []float64, []float64) {
	fast_lr := 1 / 3.0
	slow_lr := fast_lr / (1 + 90.0*fast_lr)
	fw := GoldenSectionSearch(func(f float64) float64 {
		_, _, e := rollLES(times, weights, f)
		return e
	}, slow_lr, fast_lr, 1e-4)
	fc := GoldenSectionSearch(func(f float64) float64 {
		_, _, e := rollLES(times, cals, f)
		return e
	}, slow_lr, fast_lr, 1e-4)
	w, dwdt, _ := rollLES(times, weights, fw)
	c, _, _ := rollLES(times, cals, fc)
	tdee := []float64{}
	for i := range w {
		tdee = append(tdee, c[i]-dwdt[i]*CAL_FAT)
	}
	return w, dwdt, tdee
}

func GoldenSectionSearch(f func(float64) float64, a, b, tol float64) float64 {
	invphi := (math.Sqrt(5.0) - 1.0) / 2.0
	for math.Abs(b-a) > tol {
		c := b - (b-a)*invphi
		d := a + (b-a)*invphi
		if f(c) < f(d) {
			b = d
		} else {
			a = c
		}
	}
	return (b + a) / 2
}

type LES struct {
	t1 float64
	t2 float64
	x1 float64
	x2 float64
	lr float64
}

const LES_GAP = 7.0

func NewLES(t, x, dxdt, lr float64) LES {
	return LES{t1: t - 1, t2: t - 1 - LES_GAP, x1: x, x2: x - LES_GAP*dxdt, lr: lr}
}
func (s *LES) Feed(t, x float64) {
	flr := s.lr
	slr := flr / (1.0 + LES_GAP*flr)
	s.x1 += flr * (x - s.x1)
	s.x2 += slr * (x - s.x2)
	s.t1 += flr * (t - s.t1)
	s.t2 += slr * (t - s.t2)
}
func (s *LES) Pred(t float64) float64 {
	mx := (s.x1 + s.x2) / 2.0
	mt := (s.t1 + s.t2) / 2.0
	dxdt := (s.x1 - s.x2) / (s.t1 - s.t2)
	return mx + dxdt*(t-mt)
}
func rollLES(T []float64, X []float64, lr float64) ([]float64, []float64, float64) {
	les := NewLES(T[0], X[0], 0.0, lr)
	err := 0.0
	Xhat := []float64{}
	DXhat := []float64{}
	for i := range T {
		t := T[i]
		x := X[i]
		et := les.Pred(t) - x
		err += et * et
		les.Feed(t, x)
		Xhat = append(Xhat, les.Pred(t))
		DXhat = append(DXhat, les.Pred(t+0.5)-les.Pred(t-0.5))
	}
	return Xhat, DXhat, err
}
func TDEE_V1(T []float64, W []float64, Cal []float64, lr float64) ([]float64, []float64) {
	ta := T[0] - 1/math.Sqrt(lr*(1-lr))
	wa := W[0]
	ca := Cal[0]
	twa := ta * wa
	tca := ta * ca
	tta := ta * ta
	tdee := ca
	tdee_var := 0.0 //tdee * tdee * 0.2 * 0.2
	K := CAL_FAT    // cal / (WUNIT * day)
	tdee_v := []float64{}
	tdee_sd := []float64{}
	for i := range T {
		t, w, c := T[i], W[i], Cal[i]
		ta += lr * (t - ta)
		wa += lr * (w - wa)
		ca += lr * (c - ca)
		tta += lr * (t*t - tta)
		twa += lr * (t*w - twa)
		tca += lr * (t*c - tca)
		t_var := tta - ta*ta
		dwdt := (twa - ta*wa) / t_var
		dcdt := (tca - ta*ca) / t_var
		tdee_inst := ca + dcdt*(t-ta) - dwdt*K
		delta_tdee := tdee_inst - tdee
		tdee_var += (delta_tdee*delta_tdee - tdee_var) * lr
		tdee += delta_tdee * lr
		tdee_v = append(tdee_v, tdee)
		tdee_sd = append(tdee_sd, math.Sqrt(tdee_var))
	}
	return tdee_v, tdee_sd
}
