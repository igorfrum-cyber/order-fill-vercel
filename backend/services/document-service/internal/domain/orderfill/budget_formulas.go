package orderfill

import (
	"go/ast"
	"go/parser"
	"go/token"
	"math"
	"regexp"
	"strconv"
	"strings"

	"order-fill/backend/services/document-service/internal/domain/normalize"
	"order-fill/backend/services/document-service/internal/domain/spreadsheet"
)

var budgetReference = regexp.MustCompile(`\$?([A-Z]+)\$?([0-9]+)`)
var budgetAggregate = regexp.MustCompile(`(?i)^(SUM|SUMPRODUCT)\(([^()]*)\)$`)
var budgetRound = regexp.MustCompile(`(?i)^ROUND\((.*),\s*([0-9]+)\)$`)

// BudgetCellPrice evaluates the same restricted arithmetic as the main oracle;
// unsupported Excel functions retain the supplier's numeric cache.
func BudgetCellPrice(sheet spreadsheet.Sheet, row, column int) (float64, bool) {
	formulas := map[[2]int]string{}
	if f, ok := sheet.(spreadsheet.Formulated); ok {
		for _, formula := range f.Formulas() {
			formulas[[2]int{formula.Row, formula.Column}] = formula.Text
		}
	}
	return budgetCellPrice(sheet, row, column, formulas, map[[2]int]bool{})
}

func budgetCellPrice(sheet spreadsheet.Sheet, row, column int, formulas map[[2]int]string, seen map[[2]int]bool) (float64, bool) {
	key := [2]int{row, column}
	if seen[key] {
		return 0, false
	}
	formula := formulas[key]
	if formula == "" {
		return normalize.ParseNumber(sheet.Value(row, column))
	}
	seen[key] = true
	defer delete(seen, key)
	fallback := func() (float64, bool) { return normalize.ParseNumber(sheet.Value(row, column)) }
	if parts := budgetAggregate.FindStringSubmatch(formula); len(parts) > 0 {
		ranges := [][]float64{}
		for part := range strings.SplitSeq(parts[2], ",") {
			refs := strings.Split(strings.ReplaceAll(strings.TrimSpace(part), "$", ""), ":")
			if len(refs) > 2 {
				return fallback()
			}
			a := budgetReference.FindStringSubmatch(refs[0])
			b := a
			if len(refs) == 2 {
				b = budgetReference.FindStringSubmatch(refs[1])
			}
			if len(a) != 3 || len(b) != 3 {
				return fallback()
			}
			ar, _ := strconv.Atoi(a[2])
			br, _ := strconv.Atoi(b[2])
			ac := spreadsheet.ParseColumnName(a[1])
			bc := spreadsheet.ParseColumnName(b[1])
			if br < ar || bc < ac || (br-ar+1)*(bc-ac+1) > 100000 {
				return fallback()
			}
			values := []float64{}
			for r := ar; r <= br; r++ {
				for c := ac; c <= bc; c++ {
					value, _ := budgetCellPrice(sheet, r, c, formulas, seen)
					values = append(values, value)
				}
			}
			ranges = append(ranges, values)
		}
		total := 0.0
		if strings.EqualFold(parts[1], "SUM") {
			for _, values := range ranges {
				for _, value := range values {
					total += value
				}
			}
			return total, true
		}
		for i := range len(ranges[0]) {
			product := 1.0
			for _, values := range ranges {
				if len(values) != len(ranges[0]) {
					return fallback()
				}
				product *= values[i]
			}
			total += product
		}
		return total, true
	}
	round := budgetRound.FindStringSubmatch(formula)
	expression := formula
	if len(round) > 0 {
		expression = round[1]
	}
	invalid := false
	expression = budgetReference.ReplaceAllStringFunc(expression, func(ref string) string {
		parts := budgetReference.FindStringSubmatch(ref)
		r, _ := strconv.Atoi(parts[2])
		c := spreadsheet.ParseColumnName(parts[1])
		value := 0.0
		if sheet.Value(r, c) != "" || formulas[[2]int{r, c}] != "" {
			var ok bool
			value, ok = budgetCellPrice(sheet, r, c, formulas, seen)
			if !ok {
				invalid = true
			}
		}
		return "(" + strconv.FormatFloat(value, 'f', -1, 64) + ")"
	})
	expression = strings.ReplaceAll(expression, "%", "/100")
	if invalid {
		return fallback()
	}
	for _, char := range expression {
		if !strings.ContainsRune("0123456789. \t\n()+*/-", char) {
			return fallback()
		}
	}
	node, err := parser.ParseExpr(expression)
	if err != nil {
		return 0, false
	}
	value, ok := budgetArithmetic(node)
	if !ok || math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, false
	}
	if len(round) > 0 {
		digits, _ := strconv.Atoi(round[2])
		scale := math.Pow10(digits)
		value = math.Floor((value+2.220446049250313e-16)*scale+0.5) / scale
	}
	return value, true
}

func budgetArithmetic(node ast.Expr) (float64, bool) {
	switch n := node.(type) {
	case *ast.BasicLit:
		value, err := strconv.ParseFloat(n.Value, 64)
		return value, err == nil
	case *ast.ParenExpr:
		return budgetArithmetic(n.X)
	case *ast.UnaryExpr:
		value, ok := budgetArithmetic(n.X)
		if n.Op == token.SUB {
			return -value, ok
		}
		return value, ok && n.Op == token.ADD
	case *ast.BinaryExpr:
		a, ok := budgetArithmetic(n.X)
		b, other := budgetArithmetic(n.Y)
		if !ok || !other {
			return 0, false
		}
		switch n.Op {
		case token.ADD:
			return a + b, true
		case token.SUB:
			return a - b, true
		case token.MUL:
			return a * b, true
		case token.QUO:
			return a / b, true
		}
	}
	return 0, false
}
