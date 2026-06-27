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

// WeiLin 数据 SQL 使用 INSERT OR REPLACE，字段值用单引号括起
// 格式示例：
// INSERT OR REPLACE INTO "tag_groups" ("id_index", "name", ...) VALUES (1, '人物', ...);
// INSERT OR REPLACE INTO "tag_subgroups" ("id_index", "group_id", "name", ...) VALUES (1, 1, '对象', ...);
// INSERT OR REPLACE INTO "tag_tags" ("id_index", "subgroup_id", "text", "desc", ...) VALUES (1, 1, '1girl', '1女孩', ...);

var (
	reGroups    = regexp.MustCompile(`INTO\s+"tag_groups"\s+\([^)]*\)\s+VALUES\s*\(([^)]+)\)`)
	reSubgroups = regexp.MustCompile(`INTO\s+"tag_subgroups"\s+\([^)]*\)\s+VALUES\s*\(([^)]+)\)`)
	reTags      = regexp.MustCompile(`INTO\s+"tag_tags"\s+\([^)]*\)\s+VALUES\s*\(([^)]+)\)`)
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("用法: go run scripts/seed/import_weilin.go <sql文件路径>")
		fmt.Println("示例: go run scripts/seed/import_weilin.go ~/WeiLin/tags_2025_03_31.sql")
		os.Exit(1)
	}

	vc := conf.GetConfig()
	data := infra.NewData(vc, infra.NewCustomRedisClient(infra.NewRedisClient(vc)))

	// AutoMigrate 新表
	if err := data.DB.AutoMigrate(&model.SliceType{}, &model.PromptSlice{}); err != nil {
		panic(err)
	}

	file, err := os.Open(os.Args[1])
	if err != nil {
		panic(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024) // 10MB buffer for large SQL

	// 临时存储：源 ID → DB ID 映射
	groupIDMap := make(map[int]uint)    // tag_groups.id_index → slice_types.id
	subgroupIDMap := make(map[int]uint) // tag_subgroups.id_index → slice_types.id

	var groupCount, subgroupCount, tagCount int

	for scanner.Scan() {
		line := scanner.Text()

		// 解析 tag_groups
		if m := reGroups.FindStringSubmatch(line); m != nil {
			fields := parseValues(m[1])
			if len(fields) < 5 {
				continue
			}
			srcID, _ := strconv.Atoi(fields[0])
			name := unquote(fields[1])

			st := &model.SliceType{
				Name:      name,
				ParentID:  nil,
				SortOrder: srcID,
			}
			if err := data.DB.Create(st).Error; err != nil {
				fmt.Printf("警告: 跳过重复的 group %q: %v\n", name, err)
				// 查已有记录
				var existing model.SliceType
				if data.DB.Where("name = ? AND parent_id IS NULL", name).First(&existing).Error == nil {
					groupIDMap[srcID] = existing.ID
				}
				continue
			}
			groupIDMap[srcID] = st.ID
			groupCount++
		}

		// 解析 tag_subgroups
		if m := reSubgroups.FindStringSubmatch(line); m != nil {
			fields := parseValues(m[1])
			if len(fields) < 5 {
				continue
			}
			srcID, _ := strconv.Atoi(fields[0])
			groupSrcID, _ := strconv.Atoi(fields[1])
			name := unquote(fields[2])

			parentID, ok := groupIDMap[groupSrcID]
			if !ok {
				continue // 父级不存在，跳过
			}

			st := &model.SliceType{
				Name:      name,
				ParentID:  &parentID,
				SortOrder: srcID,
			}
			if err := data.DB.Create(st).Error; err != nil {
				fmt.Printf("警告: 跳过重复的 subgroup %q: %v\n", name, err)
				var existing model.SliceType
				if data.DB.Where("name = ? AND parent_id = ?", name, parentID).First(&existing).Error == nil {
					subgroupIDMap[srcID] = existing.ID
				}
				continue
			}
			subgroupIDMap[srcID] = st.ID
			subgroupCount++
		}

		// 解析 tag_tags
		if m := reTags.FindStringSubmatch(line); m != nil {
			fields := parseValues(m[1])
			if len(fields) < 5 {
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
				fmt.Printf("警告: 跳过 tag %q: %v\n", text, err)
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

// parseValues 解析 VALUES(...) 中的逗号分隔值，处理引号内逗号
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

// unquote 去除首尾单引号
func unquote(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && s[0] == '\'' && s[len(s)-1] == '\'' {
		return s[1 : len(s)-1]
	}
	return s
}
