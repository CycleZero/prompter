package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"prompter/conf"
	"prompter/infra"
	"prompter/model"
)

var reValues = regexp.MustCompile(`VALUES\s*\((.*)\)\s*;\s*$`)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("用法: go run scripts/seed/import_weilin.go <sql文件路径>")
		os.Exit(1)
	}

	vc := conf.GetConfig()
	data := infra.NewData(vc, infra.NewCustomRedisClient(infra.NewRedisClient(vc)))

	if err := data.DB.AutoMigrate(&model.SliceType{}, &model.PromptSlice{}); err != nil {
		panic(err)
	}

	file, err := os.Open(os.Args[1])
	if err != nil {
		panic(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024)

	groupIDMap := make(map[int]uint)
	subgroupIDMap := make(map[int]uint)
	var groupCount, subgroupCount, tagCount int

	for scanner.Scan() {
		line := scanner.Text()

		m := reValues.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		fields := parseValues(m[1])

		switch {
		case strings.Contains(line, "tag_groups"):
			if len(fields) < 2 {
				continue
			}
			srcID, _ := strconv.Atoi(fields[0])
			name := unquote(fields[1])
			st := &model.SliceType{Name: name, ParentID: nil, SortOrder: srcID}
			if err := data.DB.Create(st).Error; err != nil {
				var existing model.SliceType
				if data.DB.Where("name = ? AND parent_id IS NULL", name).First(&existing).Error == nil {
					groupIDMap[srcID] = existing.ID
				}
				continue
			}
			groupIDMap[srcID] = st.ID
			groupCount++

		case strings.Contains(line, "tag_subgroups"):
			if len(fields) < 3 {
				continue
			}
			srcID, _ := strconv.Atoi(fields[0])
			groupSrcID, _ := strconv.Atoi(fields[1])
			name := unquote(fields[2])
			parentID, ok := groupIDMap[groupSrcID]
			if !ok {
				continue
			}
			st := &model.SliceType{Name: name, ParentID: &parentID, SortOrder: srcID}
			if err := data.DB.Create(st).Error; err != nil {
				var existing model.SliceType
				if data.DB.Where("name = ? AND parent_id = ?", name, parentID).First(&existing).Error == nil {
					subgroupIDMap[srcID] = existing.ID
				}
				continue
			}
			subgroupIDMap[srcID] = st.ID
			subgroupCount++

		case strings.Contains(line, "tag_tags"):
			if len(fields) < 4 {
				continue
			}
			subgroupSrcID, _ := strconv.Atoi(fields[1])
			text := unquote(fields[2])
			desc := unquote(fields[3])
			typeID, ok := subgroupIDMap[subgroupSrcID]
			if !ok {
				continue
			}
			slice := &model.PromptSlice{
				TypeID:            &typeID,
				Content:           text,
				TranslatedContent: desc,
				OriginLanguage:    model.English,
				TargetLanguage:    model.Chinese,
			}
			if err := data.DB.Create(slice).Error; err != nil {
				continue
			}
			tagCount++
		}
	}

	if err := scanner.Err(); err != nil {
		panic(err)
	}

	fmt.Printf("导入完成: %d 个一级分类, %d 个二级分类, %d 个标签\n", groupCount, subgroupCount, tagCount)
}

func parseValues(s string) []string {
	var result []string
	var current strings.Builder
	inQuote := false
	for _, c := range s {
		switch {
		case c == '\'':
			inQuote = !inQuote
			current.WriteRune(c)
		case c == ',' && !inQuote:
			result = append(result, strings.TrimSpace(current.String()))
			current.Reset()
		default:
			current.WriteRune(c)
		}
	}
	if current.Len() > 0 {
		result = append(result, strings.TrimSpace(current.String()))
	}
	return result
}

func unquote(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && s[0] == '\'' && s[len(s)-1] == '\'' {
		return s[1 : len(s)-1]
	}
	return s
}
