import { useState, useEffect, useCallback } from 'react';
import {
  Accordion,
  AccordionSummary,
  AccordionDetails,
  Chip,
  Typography,
  CircularProgress,
  TextField,
  InputAdornment,
  IconButton,
  List,
  ListItem,
} from '@mui/material';
import { Search, Clear } from '@mui/icons-material';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import type { SliceType, Slice, SearchSlice } from '../types';
import { api } from '../api/client';

// 提示词库面板属性 — onSliceClick 同时接受普通 Slice 和搜索结果 SearchSlice
interface RegionPanelProps {
  types: SliceType[];
  onSliceClick: (typeName: string, slice: Slice | SearchSlice) => void;
}

/** 片段类型树节点 — 递归渲染手风琴，展开时懒加载该类型下的片段 */
function TypeNode({
  type,
  level,
  onSliceClick,
}: {
  type: SliceType;
  level: number;
  onSliceClick: (typeName: string, slice: Slice) => void;
}) {
  const [slices, setSlices] = useState<Slice[] | null>(null);
  const [loading, setLoading] = useState(false);

  // 展开时懒加载片段列表
  const handleExpand = () => {
    if (slices === null && !loading) {
      setLoading(true);
      api
        .listSlicesByType(type.id)
        .then((res) => setSlices(res.data.list))
        .catch(() => setSlices([]))
        .finally(() => setLoading(false));
    }
  };

  return (
    <>
      <Accordion
        key={type.id}
        disableGutters
        onChange={(_event, expanded) => {
          if (expanded) handleExpand();
        }}
      >
        <AccordionSummary expandIcon={<ExpandMoreIcon />} sx={{ ml: level * 2 }}>
          <Typography variant="body2">{type.name}</Typography>
        </AccordionSummary>
        <AccordionDetails
          sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}
        >
          {loading ? (
            <CircularProgress size={20} />
          ) : slices && slices.length === 0 ? (
            <Typography variant="caption" color="text.secondary">
              暂无片段
            </Typography>
          ) : (
            slices?.map((s) => (
              <Chip
                key={s.id}
                label={s.content}
                size="small"
                variant="outlined"
                onClick={() => onSliceClick(type.name, s)}
                sx={{ cursor: 'pointer' }}
              />
            ))
          )}
        </AccordionDetails>
      </Accordion>
      {/* 递归渲染子类型 */}
      {type.children?.map((child) => (
        <TypeNode
          key={child.id}
          type={child}
          level={level + 1}
          onSliceClick={onSliceClick}
        />
      ))}
    </>
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
    <>
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
                <Search fontSize="small" />
              </InputAdornment>
            ),
            endAdornment: query ? (
              <InputAdornment position="end">
                <IconButton size="small" onClick={handleClear}>
                  <Clear fontSize="small" />
                </IconButton>
              </InputAdornment>
            ) : null,
          },
        }}
        sx={{ mb: 1 }}
      />

      {/* 搜索结果列表 */}
      {hasSearched && (
        <>
          <Typography variant="subtitle2" gutterBottom>
            搜索结果 {searching ? '' : `(${results.length})`}
          </Typography>
          {searching ? (
            <CircularProgress size={20} sx={{ my: 1 }} />
          ) : results.length === 0 ? (
            <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
              未找到匹配的标签
            </Typography>
          ) : (
            <List dense sx={{ mb: 2 }}>
              {results.map((s) => (
                <ListItem
                  key={s.id}
                  onClick={() => onSliceClick('搜索结果', s)}
                  sx={{
                    cursor: 'pointer',
                    borderRadius: 1,
                    '&:hover': { bgcolor: 'action.hover' },
                  }}
                >
                  <Chip
                    label={s.content}
                    size="small"
                    variant="filled"
                    color="primary"
                    sx={{ mr: 1 }}
                  />
                  <Typography
                    variant="caption"
                    color="text.secondary"
                    sx={{ flexGrow: 1 }}
                  >
                    {s.translated_content}
                  </Typography>
                </ListItem>
              ))}
            </List>
          )}
        </>
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
              level={0}
              onSliceClick={onSliceClick}
            />
          ))
        ))}
    </>
  );
}
