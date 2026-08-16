# The dependency graph of the pipeline, and the only copy of it.
#
# There is deliberately no `lifeplan run`: the graph belongs to the task runner
# alone, so that it is not maintained twice. Nothing
# here reaches for a GNU extension younger than make 3.81, so that this file can
# be handed to kati unchanged if it ever becomes too slow.
#
#   make            validate, then work out every project's tables, then compare
#   make validate   check the input tables of every project
#   make tables     write out/<project>/ for every project
#   make compare    write out/compare/
#   make clean      remove out/

LIFEPLAN ?= bin/lifeplan
OUT      ?= out
PROJECTS ?= projects
BASE     ?= base
GO       ?= go

# lifeplan refuses to write anywhere that is not under a directory called out
# (tools.AssertUnderOut), so anything else here would half-run and then fail.
#
# The `..` is rejected separately because the tool cleans the path before it
# looks and this cannot: `out/../elsewhere` has an element named out and is not
# under one. `make clean` runs no tool, so this is the only thing between a
# mistyped OUT and rm -rf on a directory of inputs.
ifeq ($(filter out,$(subst /, ,$(OUT))),)
$(error OUT must have a path element named out, because lifeplan writes nowhere else; got "$(OUT)")
endif
ifneq ($(filter ..,$(subst /, ,$(OUT))),)
$(error OUT must not climb out of itself with "..", since it is a directory this file removes; got "$(OUT)")
endif

# The prerequisites of every project are read out of its manifest by lifeplan
# itself, and that reading happens while this file is being parsed — before any
# recipe could have run. So the binary is built here rather than by a rule.
#
# Passing LIFEPLAN in names a binary that is already built, and then nothing is
# compiled here.
#
# It is the word and not the output that says the build failed: a go build that
# succeeds may still have written a warning, and make 3.81 has no
# $(.SHELLSTATUS) to ask instead.
#
# It is built beside the binary in use and put in its place only when its bytes
# differ, because `go build -o` rewrites its output whether or not anything
# changed. The binary's date must say when the code that works out the results
# changed, since that is what every rule below rests on: a result is as much the
# work of the code as of the tables it read. Asking go build is also a better
# answer than any list of source paths — it is the thing that knows.
#
# The scratch path carries the shell's own process id, so that two makes running
# in one tree cannot build over each other's binary or remove it.
ifeq ($(filter command line environment,$(origin LIFEPLAN)),)
BOOTSTRAP := $(shell mkdir -p $(dir $(LIFEPLAN)) && \
  NEW=$(LIFEPLAN).$$$$.new && \
  { $(GO) build -o $$NEW ./tools/lifeplan 2>&1 && \
    { cmp -s $$NEW $(LIFEPLAN) || mv -f $$NEW $(LIFEPLAN); } || echo BUILD-FAILED; }; \
  rm -f $$NEW)
ifneq ($(filter BUILD-FAILED,$(BOOTSTRAP)),)
$(error building $(LIFEPLAN) failed: $(BOOTSTRAP))
endif
endif

# Reads the clock, a second back, into TS.
#
# make 3.81 compares timestamps as whole seconds, so an input rewritten in the
# same second as the results built from it reads as no newer than them — and no
# later run separates the two seconds either, so the change is missed for good
# .
#
# So a stamp is not dated when it is written but from **before the tool read
# anything**: every recipe below reads the clock first and dates its stamp to
# that, less a second. Anything written from the moment of the read onwards is
# then strictly newer than the stamp, whole seconds or not. Dating the stamp on
# the way out instead would only cover the last second of the run — the reading
# happens earlier than that, and a table edited in between would be lost.
#
# The cost is a needless rebuild when an input was written in the second before
# the build started. That is the direction to err in: doing the work twice, and
# never reporting a plan that answers to no input.
#
# BSD and GNU date disagree about how to say "one second ago", so both are
# tried. `touch -t` is spelled the same by both.
# Each recipe marks the clock before it reads anything, and dates its stamp
# from the mark afterwards, so the commands themselves stay visible in the log.
#
# The marks are named per rule, not per project, so that the check and the
# tables of one project cannot write over each other's — nothing but the
# ordering between them would otherwise keep them apart. The comparison's mark
# begins with a dot, which no project name can (a project is $(PROJECTS)/*.tsv).
# A mark left behind by a failed run is harmless: the next run dates it again
# before reading it.
MARK  = TS=$$(date -v-1S +%Y%m%d%H%M.%S 2>/dev/null || date -d '1 second ago' +%Y%m%d%H%M.%S) && touch -t $$TS
STAMP = touch -r

PROJECT_FILES := $(sort $(wildcard $(PROJECTS)/*.tsv))
PROJECT_NAMES := $(patsubst $(PROJECTS)/%.tsv,%,$(PROJECT_FILES))
OTHER_NAMES   := $(filter-out $(BASE),$(PROJECT_NAMES))

# Counterfactuals are checked and tabled like any project and **left out of the
# comparison**. They are not choices the household could make, so a line for one
# on the assets chart would read as an option .
#
# They are here rather than nowhere because a figure that no project reproduces
# is a figure nobody measures again when the base moves.
COUNTERFACTUAL_FILES := $(sort $(wildcard $(PROJECTS)/counterfactual/*.tsv))
COUNTERFACTUAL_NAMES := $(patsubst $(PROJECTS)/%.tsv,%,$(COUNTERFACTUAL_FILES))
BUILT_NAMES          := $(PROJECT_NAMES) $(COUNTERFACTUAL_NAMES)

ifeq ($(filter $(BASE),$(PROJECT_NAMES)),)
$(error $(PROJECTS)/$(BASE).tsv is not there, and the comparison measures the others against it)
endif

# Everything a project rests on. lifeplan answers it, one bare path per line, so
# that nothing here has to know what `extends` means or how a TSV is quoted.
#
# It has to be `resolve -inputs` and not the origin column of `resolve`: a
# manifest whose slots are all overridden decides nothing and is still read, and
# a prerequisite left out is a result that quietly stops being rebuilt. The law
# tables and two of the actuals are named by no manifest at all, and -inputs
# names them too, because the code that reads them answers this .
inputs = $(sort $(shell $(LIFEPLAN) resolve -inputs $(1)))

$(foreach n,$(BUILT_NAMES),$(eval INPUTS_$(n) := $(call inputs,$(PROJECTS)/$(n).tsv)))

# An empty answer means resolve failed, and an unstated prerequisite is a table
# that silently stops being rebuilt. Stop instead.
$(foreach n,$(BUILT_NAMES),\
  $(if $(INPUTS_$(n)),,$(error lifeplan resolve $(PROJECTS)/$(n).tsv named no input)))

ALL_INPUTS := $(sort $(foreach n,$(BUILT_NAMES),$(INPUTS_$(n))))

VALIDATED := $(BUILT_NAMES:%=$(OUT)/.stamp/%.validate)
TABLED    := $(BUILT_NAMES:%=$(OUT)/%/.tables)

.PHONY: all validate tables compare clean
.DELETE_ON_ERROR:

# The comparison reads the manifests, not the tables, so nothing makes it wait
# for them. It is the default goal that says one run reports both, and it says
# so here rather than as an edge that would claim a dependency that is not there.
all: tables compare

validate: $(VALIDATED)
tables: $(TABLED)
compare: $(OUT)/compare/.compare

# A check leaves nothing behind to depend on, so the stamp is what is dated. It
# is kept out of the project's own directory, which is emptied on every rebuild.
$(OUT)/.stamp/%.validate: $(LIFEPLAN)
	@mkdir -p $(@D)
	@$(MARK) $@.mark
	$(LIFEPLAN) validate $(PROJECTS)/$*.tsv
	@$(STAMP) $@.mark $@ && rm -f $@.mark

# One command writes all of a project's tables, so one stamp stands for
# them all — and it is written inside the directory, so that deleting the
# directory is noticed as the tables being gone.
#
# The tables are built beside the directory and moved onto it. A table that this
# version no longer writes is therefore gone rather than left to be read as a
# result of this run.
$(OUT)/%/.tables: $(LIFEPLAN) | $(OUT)/.stamp/%.validate
	@rm -rf $(OUT)/$*.new
	@mkdir -p $(dir $(OUT)/.stamp/$*)
	@$(MARK) $(OUT)/.stamp/$*.tables.mark
	$(LIFEPLAN) tables -out $(OUT)/$*.new $(PROJECTS)/$*.tsv
	@$(STAMP) $(OUT)/.stamp/$*.tables.mark $(OUT)/$*.new/.tables && rm -f $(OUT)/.stamp/$*.tables.mark
	@rm -rf $(@D)
	@mv $(OUT)/$*.new $(@D)

# The scenario tables are composed here rather than in Go .
#
# `case-wife-parttime` differs from `base` in one cell of one row, and its table
# used to be a whole copy of `data/controllable/income-wife.tsv`. The columns
# were held together by `input.Shape`, which both tables go through; **the values
# were held together by nothing**, so editing the base and forgetting the copy
# would have made the scenario mean "the wife is salaried *and* the childcare
# leave input is out of date".
#
# **The composing is a task runner's job and not the pipeline's.** `extends`
# already exists for projects and adding it for tables would put a second
# inheritance in the Go, for a job `awk` finishes in twenty lines. One tool, one
# thing.
#
# The composed file is a build artifact and is not committed; the manifests name
# it, and this rule keeps it in step with the two files it is made of.
OVERLAY := $(word 1,$(wildcard tools/overlay.awk))

# `parttime` は短時間労働者として働く場合、`business` は base と同じ額を事業所得
# として受け取り必要経費が 20 万円ある場合。**base と `business` は額が同じで、
# 違うのは所得の種類と経費だけである**。
WIFE_VARIANTS := parttime business

$(WIFE_VARIANTS:%=data/controllable/scenario/income-wife-%.tsv): data/controllable/scenario/income-wife-%.tsv: data/controllable/income-wife.tsv data/controllable/scenario/income-wife-%.diff.tsv $(OVERLAY)
	awk -F'\t' -f $(OVERLAY) \
	  data/controllable/scenario/income-wife-$*.diff.tsv \
	  data/controllable/income-wife.tsv > $@.new
	@mv $@.new $@

# The prerequisites the pattern rules above cannot state, because a pattern rule
# is expanded before the stem is known. A target may be given prerequisites in
# one place and its recipe in another, so this adds to the rules above rather
# than replacing them.
$(foreach n,$(BUILT_NAMES),$(eval $(OUT)/.stamp/$(n).validate: $(INPUTS_$(n))))
$(foreach n,$(BUILT_NAMES),$(eval $(OUT)/$(n)/.tables: $(INPUTS_$(n))))

# One stamp again, and for the same reason: compare writes five files, and a
# rule naming one of them would say nothing about the other four being deleted.
#
# The checks are ordered before it and not depended on. Nothing they leave is
# read here, and nothing is lost by the edge being order-only: ALL_INPUTS is by
# construction the union of every project's inputs, so anything that would make
# a check run again is already an ordinary prerequisite of this target.
$(OUT)/compare/.compare: $(LIFEPLAN) $(ALL_INPUTS) | $(VALIDATED)
	@rm -rf $(OUT)/compare.new
	@mkdir -p $(OUT)/.stamp
	@$(MARK) $(OUT)/.stamp/.compare.mark
	$(LIFEPLAN) compare -out $(OUT)/compare.new \
	  $(PROJECTS)/$(BASE).tsv $(OTHER_NAMES:%=$(PROJECTS)/%.tsv)
	@$(STAMP) $(OUT)/.stamp/.compare.mark $(OUT)/compare.new/.compare && rm -f $(OUT)/.stamp/.compare.mark
	@rm -rf $(@D)
	@mv $(OUT)/compare.new $(@D)

clean:
	rm -rf $(OUT)
