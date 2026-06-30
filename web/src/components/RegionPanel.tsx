import { useState, useEffect } from 'react';
import {
  Tabs,
  Tab,
  Chip,
  Typography,
  Box,
  TextField,
  InputAdornment,
  CircularProgress,
  IconButton,
} from '@mui/material';
import { Search, Clear } from '@mui/icons-material';
import type { SliceType, Slice, SearchSlice } from '../types';
import { api } from '../api/client';

// 提示词库面板属性 — onSliceClick 接受普通 Slice 和搜索结果 SearchSlice
interface RegionPanelProps {
  types: SliceType[];
  onSliceClick: (typeName: string, slice: Slice | SearchSlice) => void;
}

/** 提示词库面板 — 二级 Tab 系统：一级主分类 + 二级子分类 + 标签片 */
export function RegionPanel({ types, onSliceClick }: RegionPanelProps) {
  // 当前选中的主分类（一级 Tab）
  const [activeParent, setActiveParent] = useState<number | null>(null);
  // 当前选中的子分类（二级 Tab）
  const [activeChild, setActiveChild] = useState<number | null>(null);
  // 当前子分类下的标签列表（懒加载）
  const [slices, setSlices] = useState<Slice[]>([]);
  // 标签加载状态
  const [loading, setLoading] = useState(false);
  // 搜索关键词
  const [searchQuery, setSearchQuery] = useState('');

  // 提取根级主分类（parent_id === null），按排序字段排列
  const rootTypes = types
    .filter((t) => t.parent_id === null)
    .sort((a, b) => a.sort_order - b.sort_order);

  // 提取当前主分类下的子分类（从 children 数组获取，非扁平 filter）
  const childTypes = activeParent
    ? (rootTypes.find((t) => t.id === activeParent)?.children ?? [])
        .sort((a, b) => a.sort_order - b.sort_order)
    : [];

  // 一级 Tab 自动选中第一个主分类（仅在 types 变化且尚未选中时触发）
  useEffect(() => {
    if (rootTypes.length > 0 && activeParent === null) {
      setActiveParent(rootTypes[0].id);
    }
  }, [rootTypes, activeParent]);

  // 主分类切换后：自动选中第一个子分类，并加载其标签列表
  useEffect(() => {
    if (activeParent === null) return;
    if (childTypes.length === 0) {
      setActiveChild(null);
      setSlices([]);
      return;
    }
    // 选中第一个子分类
    const firstChildId = childTypes[0].id;
    setActiveChild(firstChildId);
    // 加载该子分类下的标签
    setLoading(true);
    api.listSlicesByType(firstChildId)
      .then((res) => setSlices(res.data.list))
      .catch(() => setSlices([]))
      .finally(() => setLoading(false));
  }, [activeParent]); // eslint-disable-line react-hooks/exhaustive-deps

  // 用户点击切换二级 Tab 后重新加载标签
  useEffect(() => {
    if (activeChild === null) {
      setSlices([]);
      return;
    }
    setLoading(true);
    api.listSlicesByType(activeChild)
      .then((res) => setSlices(res.data.list))
      .catch(() => setSlices([]))
      .finally(() => setLoading(false));
  }, [activeChild]);

  // 按搜索关键词过滤标签（匹配 content 与 translated_content）
  const filteredSlices = searchQuery
    ? slices.filter(
        (s) =>
          s.content.toLowerCase().includes(searchQuery.toLowerCase()) ||
          s.translated_content?.toLowerCase().includes(searchQuery.toLowerCase()),
      )
    : slices;

  // 一级 Tab 样式
  const primaryTabSx = {
    minWidth: 'auto',
    px: 1.5,
    py: 0.5,
    fontSize: '0.8rem',
    minHeight: 36,
  };

  // 二级 Tab 样式（字号略小）
  const secondaryTabSx = {
    minWidth: 'auto',
    px: 1.5,
    py: 0.5,
    fontSize: '0.75rem',
    minHeight: 32,
  };

  // 标签片样式
  const chipSx = {
    fontSize: '0.82rem',
    m: 0.25,
    cursor: 'pointer',
    borderColor: '#c5cae9',
    '&:hover': { bgcolor: '#e3f2fd', borderColor: '#1976d2' },
  };

  return (
    <Box>
      {/* 搜索栏 — 按关键词过滤当前子分类下的标签 */}
      <TextField
        size="small"
        fullWidth
        placeholder="搜索标签..."
        value={searchQuery}
        onChange={(e) => setSearchQuery(e.target.value)}
        slotProps={{
          input: {
            startAdornment: (
              <InputAdornment position="start">
                <Search fontSize="small" />
              </InputAdornment>
            ),
            endAdornment: searchQuery ? (
              <InputAdornment position="end">
                <IconButton size="small" onClick={() => setSearchQuery('')}>
                  <Clear fontSize="small" />
                </IconButton>
              </InputAdornment>
            ) : undefined,
          },
        }}
        sx={{ mb: 1 }}
      />

      {/* 一级 Tab：主分类（可滚动，隐藏滚动条） */}
      {rootTypes.length > 0 && (
        <Box
          sx={{
            borderBottom: 1,
            borderColor: 'divider',
            mb: 0.5,
            overflow: 'auto',
            '&::-webkit-scrollbar': { display: 'none' },
          }}
        >
          <Tabs
            value={activeParent ?? rootTypes[0].id}
            onChange={(_, v) => setActiveParent(v)}
            variant="scrollable"
            scrollButtons="auto"
            sx={{ minHeight: 36 }}
          >
            {rootTypes.map((t) => (
              <Tab key={t.id} label={t.name} value={t.id} sx={primaryTabSx} />
            ))}
          </Tabs>
        </Box>
      )}

      {/* 二级 Tab：子分类（可滚动，隐藏滚动条） */}
      {childTypes.length > 0 && (
        <Box
          sx={{
            borderBottom: 1,
            borderColor: 'divider',
            mb: 1,
            overflow: 'auto',
            '&::-webkit-scrollbar': { display: 'none' },
          }}
        >
          <Tabs
            value={activeChild ?? childTypes[0].id}
            onChange={(_, v) => setActiveChild(v)}
            variant="scrollable"
            scrollButtons="auto"
            sx={{ minHeight: 32 }}
          >
            {childTypes.map((t) => (
              <Tab key={t.id} label={t.name} value={t.id} sx={secondaryTabSx} />
            ))}
          </Tabs>
        </Box>
      )}

      {/* 标签片区域：加载中 / 空状态 / 标签列表 */}
      {loading ? (
        <Box sx={{ display: 'flex', justifyContent: 'center', py: 2 }}>
          <CircularProgress size={24} />
        </Box>
      ) : filteredSlices.length === 0 ? (
        <Typography variant="body2" color="text.secondary" sx={{ py: 1 }}>
          暂无标签
        </Typography>
      ) : (
        <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0 }}>
          {filteredSlices.map((s) => (
            <Chip
              key={s.id}
              label={s.content}
              size="medium"
              variant="outlined"
              onClick={() =>
                onSliceClick(
                  childTypes.find((c) => c.id === activeChild)?.name ?? '',
                  s,
                )
              }
              sx={chipSx}
            />
          ))}
        </Box>
      )}
    </Box>
  );
}
