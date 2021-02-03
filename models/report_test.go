package models

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"gorm.io/gorm"

	"github.com/covergates/covergates/core"
)

// nolint:unused
type MockCoverReport struct {
	Name string
}

type reportSlice []*core.Report
type coverageSlice []*core.CoverageReport

func (s reportSlice) Len() int { return len(s) }
func (s reportSlice) Less(i, j int) bool {
	a, b := s[i], s[j]
	k1, k2 := a.ReportID+a.Commit, b.ReportID+b.Commit
	return k1 < k2
}
func (s reportSlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s coverageSlice) Len() int           { return len(s) }
func (s coverageSlice) Less(i, j int) bool { return s[i].Type < s[j].Type }
func (s coverageSlice) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

func testExpectReports(t *testing.T, expect, results reportSlice) {
	now := time.Now()
	for _, report := range results {
		report.CreatedAt = now
		sort.Sort(coverageSlice(report.Coverages))
	}
	for _, report := range expect {
		report.CreatedAt = now
		sort.Sort(coverageSlice(report.Coverages))
	}
	sort.Sort(results)
	sort.Sort(expect)
	if diff := cmp.Diff(expect, results); diff != "" {
		t.Fatal(diff)
	}
}

func TestReportStoreUpload(t *testing.T) {
	ctrl, service := getDatabaseService(t)
	defer ctrl.Finish()
	store := &ReportStore{DB: service}

	id := "TestReportStoreUpload"
	reports := reportSlice{
		{
			ReportID: id + "1",
			Commit:   "commit1",
			Coverages: []*core.CoverageReport{
				{
					Type: core.ReportPerl,
				},
			},
		},
		{
			ReportID: id + "1",
			Commit:   "commit1",
			Coverages: []*core.CoverageReport{
				{
					Type: core.ReportGo,
				},
			},
		},
		{
			ReportID: id + "1",
			Commit:   "commit1",
			Coverages: []*core.CoverageReport{
				{
					Type: core.ReportPerl,
				},
			},
		},
		{
			ReportID: id + "2",
			Commit:   "commit1",
			Coverages: []*core.CoverageReport{
				{
					Type: core.ReportPerl,
				},
				{
					Type: core.ReportGo,
				},
			},
		},
		{
			ReportID: id + "3",
			Commit:   "commit1",
			Coverages: []*core.CoverageReport{
				{
					Type: core.ReportPerl,
				},
				{
					Type: core.ReportPerl,
				},
			},
		},
	}

	expects := reportSlice{
		{
			ReportID: id + "1",
			Commit:   "commit1",
			Coverages: []*core.CoverageReport{
				{
					Type: core.ReportPerl,
				},
			},
		},
		{
			ReportID: id + "1",
			Commit:   "commit1",
			Coverages: []*core.CoverageReport{
				{
					Type: core.ReportPerl,
				},
				{
					Type: core.ReportGo,
				},
			},
		},
		{
			ReportID: id + "1",
			Commit:   "commit1",
			Coverages: []*core.CoverageReport{
				{
					Type: core.ReportPerl,
				},
				{
					Type: core.ReportGo,
				},
			},
		},
		{
			ReportID: id + "2",
			Commit:   "commit1",
			Coverages: []*core.CoverageReport{
				{
					Type: core.ReportPerl,
				},
				{
					Type: core.ReportGo,
				},
			},
		},
		{
			ReportID: id + "3",
			Commit:   "commit1",
			Coverages: []*core.CoverageReport{
				{
					Type: core.ReportPerl,
				},
			},
		},
	}

	for i, report := range reports {
		if err := store.Upload(report); err != nil {
			t.Fatal(err)
		}
		var results []*Report

		store.DB.Session().Preload("Coverages").Where(
			&Report{ReportID: report.ReportID, Commit: report.Commit},
		).Find(&results)

		if len(results) != 1 {
			t.Fatal()
		}
		result := results[0].ToCoreReport()
		sort.Sort(coverageSlice(result.Coverages))
		expect := expects[i]
		sort.Sort(coverageSlice(expect.Coverages))
		now := time.Now()
		result.CreatedAt, expect.CreatedAt = now, now
		if diff := cmp.Diff(expect, result); diff != "" {
			t.Fatal(diff)
		}
	}
}

func TestReportUploadOverwrite(t *testing.T) {
	const reportID = "testReportUploadOverwrite"
	ctrl, db := getDatabaseService(t)
	defer ctrl.Finish()
	store := &ReportStore{DB: db}

	reports := reportSlice{
		{
			ReportID: reportID,
			Commit:   "commit",
			Coverages: []*core.CoverageReport{
				{
					Type: core.ReportGo,
					Files: []*core.File{
						{
							Name:              "test.go",
							StatementCoverage: 0.4,
						},
					},
					StatementCoverage: 0.4,
				},
			},
		},
		{
			ReportID: reportID,
			Commit:   "commit",
			Coverages: []*core.CoverageReport{
				{
					Type: core.ReportGo,
					Files: []*core.File{
						{
							Name:              "test.go",
							StatementCoverage: 0.5,
						},
					},
					StatementCoverage: 0.5,
				},
			},
		},
	}

	for _, report := range reports {
		if err := store.Upload(report); err != nil {
			t.Fatal(err)
		}
	}

	report, err := store.Find(&core.Report{ReportID: reportID, Commit: "commit"})
	if err != nil {
		t.Fatal(err)
	}
	report.CreatedAt = reports[1].CreatedAt
	if diff := cmp.Diff(reports[1], report); diff != "" {
		t.Fatal(diff)
	}
}

func TestReportUploadReference(t *testing.T) {
	const reportID = "TestReportUploadReference"
	ctrl, db := getDatabaseService(t)
	defer ctrl.Finish()
	store := &ReportStore{DB: db}

	testReferenceCount := func(t *testing.T, db core.DatabaseService, ref *Reference, expect int) {
		var refs []*Reference
		if err := db.Session().Find(&refs, ref).Error; err != nil {
			t.Fatal(err)
		}
		if len(refs) != expect {
			t.Fatalf("new reference count %d not match %d", len(refs), expect)
		}
	}

	t.Run("should reuse reference when report update", func(t *testing.T) {
		report1 := &core.Report{
			Commit:    "abc",
			ReportID:  reportID,
			Reference: "master",
		}
		if err := store.Upload(report1); err != nil {
			t.Fatal(err)
		}
		if err := store.Upload(report1); err != nil {
			t.Fatal(err)
		}
		testReferenceCount(t, db, &Reference{ReportID: reportID, Name: "master"}, 1)
	})

	t.Run("should reuse reference for new report", func(t *testing.T) {
		report2 := &core.Report{
			Commit:    "edf",
			ReportID:  reportID,
			Reference: "master",
		}
		if err := store.Upload(report2); err != nil {
			t.Fatal(err)
		}
		testReferenceCount(t, db, &Reference{ReportID: reportID, Name: "master"}, 1)
		ref := &Reference{ReportID: reportID, Name: "master"}
		db.Session().Preload("Reports").First(ref, ref)
		if len(ref.Reports) != 2 {
			t.Fatal("should have 2 reports relate to master")
		}
	})
}

func TestReportFind(t *testing.T) {
	ctrl, service := getDatabaseService(t)
	defer ctrl.Finish()
	store := &ReportStore{DB: service}
	id := "TestReportFind"
	reports := []*core.Report{
		{
			ReportID:  id + "1",
			Reference: "master",
			Commit:    "commit1",
			Coverages: []*core.CoverageReport{
				{
					Type: core.ReportPerl,
				},
			},
		},
		{
			ReportID:  id + "1",
			Reference: "master",
			Commit:    "commit2",
			Coverages: []*core.CoverageReport{
				{
					Type: core.ReportPerl,
				},
			},
		},
		{
			ReportID:  id + "2",
			Reference: "branch1",
			Commit:    "commit1",
			Coverages: []*core.CoverageReport{
				{
					Type: core.ReportGo,
				},
			},
		},
		{
			ReportID:  id + "2",
			Reference: "branch2",
			Commit:    "commit2",
			Coverages: []*core.CoverageReport{
				{
					Type: core.ReportGo,
				},
			},
		},
		{
			ReportID:  id + "2",
			Reference: "branch2",
			Commit:    "commit3",
			Coverages: []*core.CoverageReport{
				{
					Type: core.ReportGo,
				},
			},
		},
		{
			ReportID:  id + "2",
			Reference: "master",
			Commit:    "commit4",
			Coverages: []*core.CoverageReport{
				{
					Type: core.ReportGo,
				},
			},
		},
	}

	for _, report := range reports {
		if err := store.Upload(report); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("should find latest created", func(t *testing.T) {
		rst, err := store.Find(&core.Report{
			ReportID: id + "1",
		})
		if err != nil {
			t.Fatal(err)
		}
		if rst.Commit != "commit2" {
			t.Fail()
		}
	})

	t.Run("should find with reference", func(t *testing.T) {
		queries := []*core.Report{
			{ReportID: id + "2", Reference: "branch1"},
			{ReportID: id + "2", Reference: "branch2"},
		}
		expects := []*core.Report{
			{Commit: "commit1", Reference: "branch1"},
			{Commit: "commit3", Reference: "branch2"},
		}

		for i, query := range queries {
			rst, err := store.Find(query)
			if err != nil {
				t.Log(query)
				t.Fatal(err)
			}
			expect := expects[i]
			if rst.Commit != expect.Commit || rst.Reference != expect.Reference {
				t.Fail()
			}
		}
	})

	t.Run("should not found report with reference and empty report id", func(t *testing.T) {
		rst, err := store.Find(&core.Report{Reference: "master"})
		if err == nil || !errors.Is(err, gorm.ErrRecordNotFound) {
			t.Log(rst)
			t.Fatal(err)
		}
	})

	t.Run("should return error for non existing reference", func(t *testing.T) {
		rst, err := store.Find(&core.Report{ReportID: id + "2", Reference: "fake-branch"})
		if err == nil || !errors.Is(err, gorm.ErrRecordNotFound) {
			t.Log(rst)
			t.Fatal(err)
		}
	})
}

func TestReportFinds(t *testing.T) {
	queryString := func(query *core.Report) string {
		fields := make([]string, 0)
		if query.Commit != "" {
			fields = append(fields, "commit="+query.Commit)
		}
		if query.Reference != "" {
			fields = append(fields, "reference="+query.Reference)
		}
		if query.ReportID != "" {
			fields = append(fields, "report_id="+query.ReportID)
		}
		return strings.Join(fields, ",")
	}

	ctrl, db := getDatabaseService(t)
	defer ctrl.Finish()
	store := &ReportStore{DB: db}
	id := "TestReportFinds"
	reports := reportSlice{
		{
			ReportID: id + "1",
			Commit:   "commit1",
			Coverages: []*core.CoverageReport{
				{
					Type: core.ReportPerl,
				},
			},
		},
		{
			ReportID: id + "2",
			Commit:   "commit1",
			Coverages: []*core.CoverageReport{
				{
					Type: core.ReportPerl,
				},
				{
					Type: core.ReportGo,
				},
			},
		},
		{
			ReportID: id + "2",
			Commit:   "commit2",
			Coverages: []*core.CoverageReport{
				{
					Type: core.ReportGo,
				},
			},
		},
		{
			ReportID:  id + "3",
			Commit:    "commit1",
			Reference: "master",
			Coverages: []*core.CoverageReport{
				{
					Type: core.ReportGo,
				},
			},
		},
		{
			ReportID:  id + "3",
			Commit:    "commit2",
			Reference: "master",
			Coverages: []*core.CoverageReport{
				{
					Type: core.ReportPerl,
				},
				{
					Type: core.ReportGo,
				},
			},
		},
	}

	queries := reportSlice{
		{ReportID: id + "1"},
		{ReportID: id + "2", Commit: "commit1"},
		{ReportID: id + "3", Reference: "master"},
	}

	expects := []reportSlice{
		{
			{
				ReportID: id + "1",
				Commit:   "commit1",
				Coverages: []*core.CoverageReport{
					{
						Type: core.ReportPerl,
					},
				},
			},
		},
		{
			{
				ReportID: id + "2",
				Commit:   "commit1",
				Coverages: []*core.CoverageReport{
					{
						Type: core.ReportPerl,
					},
					{
						Type: core.ReportGo,
					},
				},
			},
		},
		{
			{
				ReportID:  id + "3",
				Commit:    "commit1",
				Reference: "master",
				Coverages: []*core.CoverageReport{
					{
						Type: core.ReportGo,
					},
				},
			},
			{
				ReportID:  id + "3",
				Commit:    "commit2",
				Reference: "master",
				Coverages: []*core.CoverageReport{
					{
						Type: core.ReportPerl,
					},
					{
						Type: core.ReportGo,
					},
				},
			},
		},
	}

	for _, report := range reports {
		if err := store.Upload(report); err != nil {
			t.Fatal(err)
		}
	}

	base := 0
	for i, query := range queries[base:] {
		t.Run(queryString(query), func(t *testing.T) {
			expect := expects[i+base]
			results, err := store.Finds(query)
			if err != nil {
				t.Fatal(err)
			}
			testExpectReports(t, expect, reportSlice(results))
		})
	}
}

func TestReportList(t *testing.T) {
	ctrl, db := getDatabaseService(t)
	defer ctrl.Finish()
	store := &ReportStore{DB: db}
	id := "TestReportList"
	reports := reportSlice{
		{
			ReportID: id + "1",
			Commit:   "commit1",
			Coverages: []*core.CoverageReport{
				{
					Type: core.ReportPerl,
				},
			},
		},
		{
			ReportID: id + "1",
			Commit:   "commit1",
			Coverages: []*core.CoverageReport{
				{
					Type: core.ReportGo,
				},
			},
		},
		{
			ReportID: id + "2",
			Commit:   "commit1",
			Coverages: []*core.CoverageReport{
				{
					Type: core.ReportGo,
				},
			},
			Reference: "master",
		},
		{
			ReportID: id + "2",
			Commit:   "commit2",
			Coverages: []*core.CoverageReport{
				{
					Type: core.ReportGo,
				},
			},
			Reference: "master",
		},
	}

	queries := [][]string{
		{id + "1", "commit1"},
		{id + "2", "commit2"},
		{id + "2", "master"},
	}

	expectations := []reportSlice{
		{
			{
				ReportID: id + "1",
				Commit:   "commit1",
				Files:    []string{},
				Coverages: []*core.CoverageReport{
					{
						Type: core.ReportPerl,
					},
					{
						Type: core.ReportGo,
					},
				},
			},
		},
		{
			{
				ReportID: id + "2",
				Commit:   "commit2",
				Files:    []string{},
				Coverages: []*core.CoverageReport{
					{
						Type: core.ReportGo,
					},
				},
			},
		},
		{
			{
				ReportID: id + "2",
				Commit:   "commit1",
				Files:    []string{},
				Coverages: []*core.CoverageReport{
					{
						Type: core.ReportGo,
					},
				},
				Reference: "master",
			},
			{
				ReportID: id + "2",
				Commit:   "commit2",
				Files:    []string{},
				Coverages: []*core.CoverageReport{
					{
						Type: core.ReportGo,
					},
				},
				Reference: "master",
			},
		},
	}

	for _, report := range reports {
		if err := store.Upload(report); err != nil {
			t.Fatal(err)
		}
	}
	base := 0
	for i, query := range queries[base:] {
		t.Run(fmt.Sprintf("%s,%s", query[0], query[1]), func(t *testing.T) {
			result, err := store.List(query[0], query[1])
			if err != nil {
				t.Fatal(err)
			}
			testExpectReports(t, expectations[i+base], reportSlice(result))
		})
	}
}

func TestReportUploadFiles(t *testing.T) {
	ctrl, service := getDatabaseService(t)
	defer ctrl.Finish()
	store := &ReportStore{
		DB: service,
	}
	m := &core.Report{
		ReportID: "test_upload_files",
		Commit:   "test_upload_files",
		Files:    []string{"a", "b", "c"},
	}
	if err := store.Upload(m); err != nil {
		t.Error(err)
		return
	}
	report, err := store.Find(&core.Report{
		ReportID: m.ReportID,
		Commit:   m.Commit,
	})
	if err != nil {
		t.Error(err)
		return
	}
	if !reflect.DeepEqual(report.Files, m.Files) {
		t.Fail()
	}
}

func TestReportComment(t *testing.T) {
	ctrl, service := getDatabaseService(t)
	defer ctrl.Finish()
	store := &ReportStore{
		DB: service,
	}

	report := &core.Report{
		ReportID: "ABCD",
	}

	if err := store.CreateComment(report, &core.ReportComment{}); err == nil {
		t.Fail()
	}

	if err := store.CreateComment(report, &core.ReportComment{Comment: 1, Number: 1}); err != nil {
		t.Fatal(err)
	}
	comment, err := store.FindComment(report, 1)
	if err != nil {
		t.Fatal(err)
	}
	if comment.Comment != 1 {
		t.Fail()
	}
	if err = store.CreateComment(report, &core.ReportComment{Comment: 2, Number: 1}); err != nil {
		t.Fatal(err)
	}
	comment, err = store.FindComment(report, 1)
	if err != nil {
		t.Fatal(err)
	}
	if comment.Comment != 2 {
		t.Fail()
	}
	if _, err := store.FindComment(report, 123); err == nil {
		t.Fail()
	}
}
