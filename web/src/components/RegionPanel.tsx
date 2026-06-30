import { useState, useEffect, useCallback } from 'react';
import {
  Box,
  Chip,
  Typography,
  CircularProgress,
  TextField,
  InputAdornment,
  IconButton,
  Collapse,
} from '@mui/material';
import { Search, Clear, ExpandMore, ExpandLess } from '@mui/icons-material';
import type { SliceType, Slice, SearchSlice } from '../types';
import { api } from '../api/client';

// 提示词库面板属性 — onSliceClick 接受普通 Slice 和搜索结果 SearchSlice
interface RegionPanelProps {
  types: SliceType[];
  onSliceClick: (typeName: string, slice: Slice | SearchSlice) => void;
}

/** 片段类型树节点 — 顶层分类可展开/收起，子分类作为区块展示标签 */
function TypeNode({
  type,
  onSliceClick,
}: {
  type: SliceType;
  onSliceClick: (typeName: string, slice: Slice) => void;
}) {
  const [expanded, setExpanded] = useState(false);
  // 按子类型 ID 缓存已加载的片段列表
  const [childSlices, setChildSlices] = useState<Record<number, Slice[]>>({});
  const [loading, setLoading] = useState(false);

  // 展开时懒加载所有子类型的片段
  const handleToggle = () => {
    if (expanded) {
      setExpanded(false);
      return;
    }
    setExpanded(true);
    if (type.children.length === 0) return;

    setLoading(true);
    // 并行加载所有子类型的片段
    const loadPromises = type.children.map((child) =>
      api
        .listSlicesByType(child.id)
        .then((res) => ({ childId: child.id, slices: res.data.list }))
        .catch(() => ({ childId: child.id, slices: [] as Slice[] })),
    );
    Promise.all(loadPromises)
      .then((results) => {
        const newSlices: Record<number, Slice[]> = {};
        for (const r of results) {
          newSlices[r.childId] = r.slices;
        }
        setChildSlices((prev) => ({ ...prev, ...newSlices }));
      })
      .finally(() => setLoading(false));
  };

  // 若无子类型，直接将本类型的切片平铺展示
  const hasChildren = type.children.length > 0;

  return (
    <Box sx={{ mb: 1 }}>
      {/* 分类标题行 — 点击展开/收起 */}
      <Box
        onClick={handleToggle}
        sx={{
          display: 'flex',
          alignItems: 'center',
          cursor: 'pointer',
          py: 0.75,
          px: 1,
          borderRadius: 1,
          '&:hover': { bgcolor: '#e3f2fd' },
          userSelect: 'none',
        }}
      >
        {hasChildren && (
          expanded ? <ExpandLess fontSize="small" sx={{ mr: 0.5, color: 'text.secondary' }} />
            : <ExpandMore fontSize="small" sx={{ mr: 0.5, color: 'text.secondary' }} />
        )}
        <Typography variant="body2" sx={{ fontWeight: 600 }}>
          {type.name}
        </Typography>
      </Box>

      {/* 展开后显示子分类区块 */}
      <Collapse in={expanded}>
        <Box sx={{ ml: 3, mt: 0.5 }}>
          {loading ? (
            <Box sx={{ display: 'flex', justifyContent: 'center', py: 1 }}>
              <CircularProgress size={16} />
            </Box>
          ) : hasChildren ? (
            // 有子类型：按子类型分组展示标签
            type.children.map((child) => {
              const slices = childSlices[child.id] || [];
              return (
                <Box key={child.id} sx={{ mb: 1.5 }}>
                  {/* 子分类标签（分隔标题） */}
                  <Typography
                    variant="caption"
                    sx={{
                      color: 'text.secondary',
                      display: 'block',
                      mb: 0.5,
                      fontWeight: 500,
                    }}
                  >
                    {child.name}
                  </Typography>
                  {/* 子分类下的片段标签 */}
                  <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
                    {slices.length === 0 && !loading ? (
                      <Typography variant="caption" color="text.disabled">
                        暂无片段
                      </Typography>
                    ) : (
                      slices.map((s) => (
                        <Chip
                          key={s.id}
                          label={s.content}
                          size="small"
                          variant="outlined"
                          onClick={(e) => {
                            e.stopPropagation();
                            onSliceClick(child.name, s);
                          }}
                          sx={{
                            cursor: 'pointer',
                            borderColor: '#c5cae9',
                            '&:hover': { bgcolor: '#e3f2fd' },
                          }}
                        />
                      ))
                    )}
                  </Box>
                </Box>
              );
            })
          ) : (
            // 无子类型：暂无内容
            <Typography variant="caption" color="text.disabled">
              暂无子分类
            </Typography>
          )}
        </Box>
      </Collapse>
    </Box>
  );
}

/** 提示词库面板 — 搜索框 + 分类树，支持全文搜索与按类型浏览两种模式 */
export function RegionPanel({ types, onSliceClick }: RegionPanelProps) {
  const [query, setQuery] = useState('');
  const [results, setResults] = useState<SearchSlice[]>([]);
  const [searching, setSearching] = useState(false);
  const [hasSearched, setHasSearched] = useState(false);

  // 执行搜索
  const doSearch = useCallback(async (q: string) => {
    if (!q.trim()) {
      setResults([]);
      setHasSearched(false);
      return;
    }
    setSearching(true);
    setHasSearched(true);
    try {
      const res = await api.searchSlices({ q: q.trim(), page_size: 50 });
      setResults(res.data.list);
    } catch {
      setResults([]);
    } finally {
      setSearching(false);
    }
  }, []);

  // 200ms 防抖搜索
  useEffect(() => {
    const timer = setTimeout(() => doSearch(query), 200);
    return () => clearTimeout(timer);
  }, [query, doSearch]);

  // 清空搜索
  const handleClear = () => {
    setQuery('');
    setResults([]);
    setHasSearched(false);
  };

  return (
    <Box>
      {/* 搜索框 */}
      <TextField
        size="small"
        fullWidth
        placeholder="搜索标签..."
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        slotProps={{
          input: {
            startAdornment: (
              <InputAdornment position="start">
                <Search fontSize="small" sx={{ color: 'text.secondary' }} />
              </InputAdornment>
            ),
            endAdornment: query ? (
              <InputAdornment position="end">
                <IconButton size="small" onClick={handleClear}>
                  <Clear fontSize="small" />
                </IconButton>
              </InputAdornment>
            ) : null,
            sx: { bgcolor: '#f8f9fa' },
          },
        }}
        sx={{ mb: 1.5 }}
      />

      {/* 搜索结果列表 */}
      {hasSearched && (
        <Box sx={{ mb: 1 }}>
          <Typography variant="subtitle2" gutterBottom>
            搜索结果 {searching ? '' : `(${results.length})`}
          </Typography>
          {searching ? (
            <Box sx={{ display: 'flex', justifyContent: 'center', py: 1 }}>
              <CircularProgress size={20} />
            </Box>
          ) : results.length === 0 ? (
            <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
              未找到匹配的标签
            </Typography>
          ) : (
            <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.75, mb: 2 }}>
              {results.map((s) => (
                <Chip
                  key={s.id}
                  label={s.content}
                  size="small"
                  variant="filled"
                  color="primary"
                  onClick={() => onSliceClick('搜索结果', s)}
                  sx={{ cursor: 'pointer' }}
                />
              ))}
            </Box>
          )}
        </Box>
      )}

      {/* 分类树（无搜索时显示） */}
      {!hasSearched &&
        (types.length === 0 ? (
          <Typography variant="body2" color="text.secondary">
            暂无提示词类型数据
          </Typography>
        ) : (
          types.map((t) => (
            <TypeNode
              key={t.id}
              type={t}
              onSliceClick={onSliceClick}
            />
          ))
        ))}
    </Box>
  );
}
