//Copyright 2013 GoGraphviz Authors
//
//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
//Unless required by applicable law or agreed to in writing, software
//distributed under the License is distributed on an "AS IS" BASIS,
//WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//See the License for the specific language governing permissions and
//limitations under the License.

package gographviz

import (
	"github.com/awalterschulze/gographviz/ast"
)

//Creates a Graph structure by analysing an Abstract Syntax Tree representing a parsed graph.
func NewAnalysedGraph(graph *ast.Graph) (*Graph, error) {
	g := NewGraph()
	if err := Analyse(graph, g); err != nil {
		return nil, err
	}
	return g, nil
}

//Analyses an Abstract Syntax Tree representing a parsed graph into a newly created graph structure Interface.
func Analyse(graph *ast.Graph, g Interface) error {
	gerr := newErrCatcher(g)
	graph.Walk(&graphVisitor{gerr})
	return gerr.getError()
}

type nilVisitor struct {
}

func (this *nilVisitor) Visit(v ast.Elem) ast.Visitor {
	return this
}

type graphVisitor struct {
	g errInterface
}

func (this *graphVisitor) Visit(v ast.Elem) ast.Visitor {
	graph, ok := v.(*ast.Graph)
	if !ok {
		return this
	}
	this.g.SetStrict(graph.Strict)
	this.g.SetDir(graph.Type == ast.DIGRAPH)
	graphName := graph.ID.String()
	this.g.SetName(graphName)
	return newStmtVisitor(this.g, graphName)
}

func newStmtVisitor(g errInterface, graphName string) *stmtVisitor {
	return &stmtVisitor{g, graphName, make(map[string]string), make(map[string]string), make(map[string]string)}
}

type stmtVisitor struct {
	g                 errInterface
	graphName         string
	currentNodeAttrs  map[string]string
	currentEdgeAttrs  map[string]string
	currentGraphAttrs map[string]string
}

func (this *stmtVisitor) Visit(v ast.Elem) ast.Visitor {
	switch s := v.(type) {
	case ast.NodeStmt:
		return this.nodeStmt(s)
	case ast.EdgeStmt:
		return this.edgeStmt(s)
	case ast.NodeAttrs:
		return this.nodeAttrs(s)
	case ast.EdgeAttrs:
		return this.edgeAttrs(s)
	case ast.GraphAttrs:
		return this.graphAttrs(s)
	case *ast.SubGraph:
		return this.subGraph(s)
	case *ast.Attr:
		return this.attr(s)
	case ast.AttrList:
		return &nilVisitor{}
	default:
		//fmt.Fprintf(os.Stderr, "unknown stmt %T\n", v)
	}
	return this
}

func ammend(attrs map[string]string, add map[string]string) map[string]string {
	for key, value := range add {
		if _, ok := attrs[key]; !ok {
			attrs[key] = value
		}
	}
	return attrs
}

func overwrite(attrs map[string]string, overwrite map[string]string) map[string]string {
	for key, value := range overwrite {
		attrs[key] = value
	}
	return attrs
}

func (this *stmtVisitor) nodeStmt(stmt ast.NodeStmt) ast.Visitor {
	attrs := ammend(stmt.Attrs.GetMap(), this.currentNodeAttrs)
	this.g.AddNode(this.graphName, stmt.NodeID.String(), attrs)
	return &nilVisitor{}
}

func (this *stmtVisitor) edgeStmt(stmt ast.EdgeStmt) ast.Visitor {
	attrs := stmt.Attrs.GetMap()
	attrs = ammend(attrs, this.currentEdgeAttrs)
	src := stmt.Source.GetID()
	srcName := src.String()
	if stmt.Source.IsNode() {
		this.g.AddNode(this.graphName, srcName, this.currentNodeAttrs)
	}
	srcPort := stmt.Source.GetPort()
	for i := range stmt.EdgeRHS {
		directed := bool(stmt.EdgeRHS[i].Op)
		dst := stmt.EdgeRHS[i].Destination.GetID()
		dstName := dst.String()
		if stmt.EdgeRHS[i].Destination.IsNode() {
			this.g.AddNode(this.graphName, dstName, this.currentNodeAttrs)
		}
		dstPort := stmt.EdgeRHS[i].Destination.GetPort()
		this.g.AddPortEdge(srcName, srcPort.String(), dstName, dstPort.String(), directed, attrs)
		src = dst
		srcPort = dstPort
		srcName = dstName
	}
	return this
}

func (this *stmtVisitor) nodeAttrs(stmt ast.NodeAttrs) ast.Visitor {
	this.currentNodeAttrs = overwrite(this.currentNodeAttrs, ast.AttrList(stmt).GetMap())
	return &nilVisitor{}
}

func (this *stmtVisitor) edgeAttrs(stmt ast.EdgeAttrs) ast.Visitor {
	this.currentEdgeAttrs = overwrite(this.currentEdgeAttrs, ast.AttrList(stmt).GetMap())
	return &nilVisitor{}
}

func (this *stmtVisitor) graphAttrs(stmt ast.GraphAttrs) ast.Visitor {
	attrs := ast.AttrList(stmt).GetMap()
	for key, value := range attrs {
		this.g.AddAttr(this.graphName, key, value)
	}
	this.currentGraphAttrs = overwrite(this.currentGraphAttrs, attrs)
	return &nilVisitor{}
}

func (this *stmtVisitor) subGraph(stmt *ast.SubGraph) ast.Visitor {
	subGraphName := stmt.ID.String()
	this.g.AddSubGraph(this.graphName, subGraphName, this.currentGraphAttrs)
	return newStmtVisitor(this.g, subGraphName)
}

func (this *stmtVisitor) attr(stmt *ast.Attr) ast.Visitor {
	this.g.AddAttr(this.graphName, stmt.Field.String(), stmt.Value.String())
	return this
}
