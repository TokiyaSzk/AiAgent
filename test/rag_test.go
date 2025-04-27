package test

import (
	"context"
	"log"
	"testing"

	"github.com/aiagent/pkg/base"
	"github.com/aiagent/pkg/rag"
	"github.com/aiagent/pkg/sql"
	"github.com/stretchr/testify/assert"
)

func TestInitEmbedder(t *testing.T) {
	embedder, err := rag.InitEmbedder()
	assert.NoError(t, err, "初始化嵌入模型应成功")
	assert.NotNil(t, embedder, "嵌入模型不应为空")
}

func TestEmbedText(t *testing.T) {
	ctx := context.Background()
	embedder, err := rag.InitEmbedder()
	assert.NoError(t, err, "初始化嵌入模型应成功")

	testText := "这是一个测试文本"
	embedding, err := rag.EmbedText(ctx, testText, embedder)
	assert.NoError(t, err, "生成文本嵌入应成功")
	assert.NotNil(t, embedding, "嵌入向量不应为空")
	assert.Greater(t, len(embedding), 0, "嵌入向量长度应大于0")
}

func TestInsertAndRetrieveDocument(t *testing.T) {

	ctx := context.Background()
	db, err := sql.CreatePSQLClient(ctx)
	assert.NoError(t, err, "创建数据库连接应成功")
	defer db.Close()

	embedder, err := rag.InitEmbedder()
	assert.NoError(t, err, "初始化嵌入模型应成功")

	// 测试文档
	testDoc := "舞萌DX（又称maimai）是有日本世嘉公司制作的世界人气音乐游戏。它拥有国服、日服、国际服三个区服。"

	// 插入文档
	err = rag.InsertDocument(ctx, db, testDoc, embedder)
	assert.NoError(t, err, "插入文档应成功")

	// 嵌入查询文本
	queryText := "舞萌介绍"
	queryVec, err := rag.EmbedText(ctx, queryText, embedder)
	assert.NoError(t, err, "嵌入查询文本应成功")

	// 检索相关文档
	docs, err := rag.RetrieveRelevantDocs(ctx, queryVec, 1, db)
	log.Printf("检索到的文档: %v", docs)
	assert.NoError(t, err, "检索文档应成功")
	assert.NotEmpty(t, docs, "应该至少检索到一个文档")
}

func TestInsertAndRetrieveMemory(t *testing.T) {

	ctx := context.Background()
	db, err := sql.CreatePSQLClient(ctx)
	assert.NoError(t, err, "创建数据库连接应成功")
	defer db.Close()

	embedder, err := rag.InitEmbedder()
	assert.NoError(t, err, "初始化嵌入模型应成功")

	// 测试文档
	testDoc := "你是Tokiya制作的智慧生命体"

	// 插入文档
	err = rag.InsertMemory(ctx, db, testDoc, embedder)
	assert.NoError(t, err, "插入文档应成功")

	// 嵌入查询文本
	queryText := "什么是AI？"
	queryVec, err := rag.EmbedText(ctx, queryText, embedder)
	assert.NoError(t, err, "嵌入查询文本应成功")

	// 检索相关文档
	docs, err := rag.RetrieveRelevantMemory(ctx, queryVec, 1, db)
	assert.NoError(t, err, "检索文档应成功")
	assert.NotEmpty(t, docs, "应该至少检索到一个文档")
}
func TestFloat32To64(t *testing.T) {
	f32 := []float32{1.1, 2.2, 3.3}
	f64 := rag.Float32To64(f32)

	assert.Equal(t, len(f32), len(f64), "转换后的数组长度应保持不变")
	for i := range f32 {
		assert.Equal(t, float64(f32[i]), f64[i], "转换后的值应相等")
	}
}

func TestFloat64To32(t *testing.T) {
	f64 := []float64{1.1, 2.2, 3.3}
	f32 := rag.Float64To32(f64)

	assert.Equal(t, len(f64), len(f32), "转换后的数组长度应保持不变")
	for i := range f64 {
		assert.Equal(t, float32(f64[i]), f32[i], "转换后的值应相等")
	}
}

func TestFloat64ArrayToPGVector(t *testing.T) {
	vec := []float64{1.1, 2.2, 3.3}
	str := rag.Float64ArrayToPGVector(vec)
	expected := "[1.100000,2.200000,3.300000]"
	assert.Equal(t, expected, str, "应正确转换为PGVector格式")
}

func TestRunRAG(t *testing.T) {

	ctx := context.Background()
	db, err := sql.CreatePSQLClient(ctx)
	assert.NoError(t, err, "创建数据库连接应成功")
	defer db.Close()

	embedder, err := rag.InitEmbedder()
	assert.NoError(t, err, "初始化嵌入模型应成功")

	llm, err := base.CreateLLMClient()
	assert.NoError(t, err, "创建LLM客户端应成功")

	// 测试RAG流程
	question := "什么是人工智能？"
	answer, err := rag.RunRAG(ctx, question, 3, embedder, db, llm)
	assert.NoError(t, err, "RAG流程应成功执行")
	assert.NotEmpty(t, answer, "应生成一个非空回答")
}

func TestJoinDocs(t *testing.T) {
	docs := []string{"文档1", "文档2", "文档3"}
	joined := rag.JoinDocs(docs)
	expected := "- 文档1\n- 文档2\n- 文档3\n"
	assert.Equal(t, expected, joined, "应正确连接文档")
}

func TestInsertMemoryAndRetrieveRelevantMemory(t *testing.T) {
	ctx := context.Background()
	db, err := sql.CreatePSQLClient(ctx)
	assert.NoError(t, err, "创建数据库连接应成功")
	defer db.Close()

	embedder, err := rag.InitEmbedder()
	assert.NoError(t, err, "初始化嵌入模型应成功")

	// 测试文档
	testDoc := "你是Tokiya制作的智慧生命体"

	// 插入文档
	err = rag.InsertMemory(ctx, db, testDoc, embedder)
	assert.NoError(t, err, "插入文档应成功")

	// 嵌入查询文本
	queryText := "你是谁"
	queryVec, err := rag.EmbedText(ctx, queryText, embedder)
	assert.NoError(t, err, "嵌入查询文本应成功")

	// 检索相关文档
	docs, err := rag.RetrieveRelevantMemory(ctx, queryVec, 1, db)
	assert.NoError(t, err, "检索文档应成功")
	assert.NotEmpty(t, docs, "应该至少检索到一个文档")
	assert.Contains(t, docs[0], "Tokiya", "检索到的文档应包含预期内容")

	t.Logf("检索到的相关内存: %v", docs)
}
