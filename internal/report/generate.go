package report

import (
	"dependency-confusion/tools"
	"path/filepath"
	"runtime"

	"github.com/johnfercher/maroto/pkg/color"
	"github.com/johnfercher/maroto/pkg/consts"
	"github.com/johnfercher/maroto/pkg/pdf"
	"github.com/johnfercher/maroto/pkg/props"
	log "github.com/sirupsen/logrus"
)

func Generate(target string, contents [][]string) error {
	m := pdf.NewMaroto(consts.Portrait, consts.A4)
	m.SetPageMargins(20, 10, 20)

	_, b, _, _ := runtime.Caller(0)
	basepath := filepath.Dir(b)

	m.SetFontLocation(filepath.Join(basepath, "fonts"))

	m.AddUTF8Font("CustomArial", consts.Normal, "arial_custom.ttf")
	m.AddUTF8Font("CustomArial", consts.Italic, "arial_custom.ttf")
	m.AddUTF8Font("CustomArial", consts.Bold, "arial_custom.ttf")
	m.AddUTF8Font("CustomArial", consts.BoldItalic, "arial_custom.ttf")

	m.AddUTF8Font("CustomCourier", consts.Normal, "couriercyrps.ttf")
	m.AddUTF8Font("CustomCourier", consts.Italic, "couriercyrps_inclined.ttf")
	m.AddUTF8Font("CustomCourier", consts.Bold, "couriercyrps_bold.ttf")
	m.AddUTF8Font("CustomCourier", consts.BoldItalic, "couriercyrps_boldinclined.ttf")

	m.SetDefaultFontFamily("CustomArial")

	buildHeading(m, basepath, target)
	buildPackageList(m, contents)

	projectDir := tools.GetDirectoryProject()

	err := m.OutputFileAndClose(filepath.Join(projectDir, "public", "pdfs", "report.pdf"))
	if err != nil {
		log.Errorln("⚠️ Could not save PDF:", err)
		return err
	}

	log.Infoln("PDF saved successfully")
	return nil
}

func buildHeading(m pdf.Maroto, dirPath, target string) {
	m.RegisterHeader(func() {
		m.Row(50, func() {
			m.Col(12, func() {
				err := m.FileImage(filepath.Join(dirPath, "images", "logo_dependency_confusion.jpg"), props.Rect{
					Center:  true,
					Percent: 75,
				})

				if err != nil {
					log.Errorln("Image file was not loaded 😱 - ", err)
				}
			})
		})
	})

	m.Row(20, func() {
		m.Col(12, func() {
			m.Text("Отчет об уязвимости Dependency Confusion (" + target + ")", props.Text{
				Top:    3,
				Style:  consts.Bold,
				Align:  consts.Center,
				Color:  getDarkPurpleColor(),
				Size:   18,
				Family: "CustomArial",
			})
		})
	})
}

func buildPackageList(m pdf.Maroto, contents [][]string) {
	tableHeadings := []string{"Имя реестра", "Название пакета", "Версия"}
	lightPurpleColor := getLightPurpleColor()

	m.SetBackgroundColor(getTealColor())
	m.Row(10, func() {
		m.Col(12, func() {
			m.Text("Уязвимые пакеты", props.Text{
				Top:    2,
				Size:   13,
				Color:  color.NewWhite(),
				Family: "CustomCourier",
				Style:  consts.Bold,
				Align:  consts.Center,
			})
		})
	})

	m.SetBackgroundColor(color.NewWhite())

	m.TableList(tableHeadings, contents, props.TableList{
		HeaderProp: props.TableListContent{
			Size:      12,
			GridSizes: []uint{3, 6, 3},
		},
		ContentProp: props.TableListContent{
			Size:      10,
			GridSizes: []uint{3, 6, 3},
		},
		Align:                consts.Left,
		AlternatedBackground: &lightPurpleColor,
		HeaderContentSpace:   1,
		Line:                 false,
	})

}

func getDarkPurpleColor() color.Color {
	return color.Color{
		Red:   88,
		Green: 80,
		Blue:  99,
	}
}

func getLightPurpleColor() color.Color {
	return color.Color{
		Red:   210,
		Green: 200,
		Blue:  230,
	}
}

func getTealColor() color.Color {
	return color.Color{
		Red:   3,
		Green: 166,
		Blue:  166,
	}
}
