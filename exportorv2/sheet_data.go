package exportorv2

import (
	"strings"

	"github.com/davyxu/tabtoy/exportorv2/filter"
	"github.com/davyxu/tabtoy/exportorv2/model"
	"github.com/davyxu/tabtoy/util"
)

const (
	// 信息所在的行
	DataSheetRow_FieldName = 0 // 字段名(对应proto)
	DataSheetRow_FieldType = 1 // 字段类型
	DataSheetRow_FieldMeta = 2 // 字段特性
	DataSheetRow_Comment   = 3 // 用户注释
	DataSheetRow_DataBegin = 4 // 数据开始
)

type DataSheet struct {
	*Sheet

	*DataHeader
}

func (self *DataSheet) Valid() bool {
	return self.GetCellData(0, 0) != ""
}

func dataProcessor(file *File, fieldDef *model.FieldDefine, rawValue string, node *model.Node) bool {

	//	if node.Define.Name == "Type" {
	//		a := 1
	//		a++
	//	}

	// 列表
	if fieldDef.IsRepeated {

		spliter := fieldDef.ListSpliter()

		// 使用多格子实现的repeated
		if spliter == "" {

			if _, ok := filter.ConvertValue(fieldDef, rawValue, file.TypeSet, node); !ok {
				goto ConvertError
			}

		} else {
			// 一个格子切割的repeated

			valueList := strings.Split(rawValue, spliter)

			for _, v := range valueList {

				if _, ok := filter.ConvertValue(fieldDef, v, file.TypeSet, node); !ok {
					goto ConvertError
				}
			}

		}

	} else {

		// 单值
		if cv, ok := filter.ConvertValue(fieldDef, rawValue, file.TypeSet, node); !ok {
			goto ConvertError

		} else {

			// 值重复检查
			if fieldDef.Meta.RepeatCheck && !file.checkValueRepeat(fieldDef, cv) {
				log.Errorf("found repeat value, %s raw: '%s'", fieldDef.String(), cv)
				return false
			}
		}

	}

	return true

ConvertError:

	log.Errorf("value convert error, %s raw: '%s'", fieldDef.String(), rawValue)

	return false
}

func (self *DataSheet) Export(file *File, tab *model.Table) bool {

	// 检查引导头
	if !self.ParseProtoField(self.Sheet, file.TypeSet) {
		return true
	}

	// 是否继续读行
	var readingLine bool = true

	// 遍历每一行
	for self.Row = DataSheetRow_DataBegin; readingLine; self.Row++ {

		// 第一列是空的，结束
		if self.GetCellData(self.Row, 0) == "" {
			break
		}

		record := model.NewRecord()

		// 遍历每一列
		for self.Column = 0; self.Column < len(self.headerFields); self.Column++ {

			fieldDef := self.FetchFieldDefine(self.Column)

			// 数据大于列头时, 结束这个列
			if fieldDef == nil {
				break
			}

			// #开头表示注释, 跳过
			if strings.Index(fieldDef.Name, "#") == 0 {
				continue
			}

			rawValue := self.GetCellData(self.Row, self.Column)

			// repeated的, 没有填充的, 直接跳过, 不生成数据
			if rawValue == "" && fieldDef.IsRepeated {
				continue
			}

			node := record.NewNodeByDefine(fieldDef)

			// 结构体要多添加一个节点, 处理repeated 结构体情况
			if fieldDef.Type == model.FieldType_Struct {

				node.StructRoot = true
				node = node.AddKey(fieldDef)

			}

			if !dataProcessor(file, fieldDef, rawValue, node) {
				goto ErrorStop
			}

		}

		tab.Add(record)

	}

	return true

ErrorStop:

	r, c := self.GetRC()

	log.Errorf("%s|%s(%s)", self.file.FileName, self.Name, util.ConvR1C1toA1(r, c))
	return false
}

func newDataSheet(sheet *Sheet) *DataSheet {

	return &DataSheet{
		Sheet:      sheet,
		DataHeader: newDataHeadSheet(),
	}
}