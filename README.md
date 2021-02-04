Assume two machines: 

* *editor*
* *builder*.

Acme/Edwood runs on
*editor*. Compiles happen on *builder*. This arrangement requires a
mechanism to maintain the source trees of the two machines in sync.

We do not require a general purpose mechanism like a filesystem because Acme/Edwood
can tell us whenever *editor* mutates the files. So, instead have something that operates
like this:

* one of *editor* or *builder* has the canonical source tree (i.e. performs
source code control action)

* Call the machine
containing the true source version (i.e. where source control runs) the *host* and
the other machine the *client*. 

* The *editor* / *builder* roles are orthogonal to the *host* / *client* roles -- i.e.
the *editor* can be  *host* or *client*.

* *host* vs *client* is about where SCS happens

* On SCS action, *bulk-push* the source tree from *host* to *client*. (Remember
that *host* has the true version of the source.)

* run `slurp` on the *builder*. It slurps the edits from Acme/Edwood (on the *editor*)
to the *builder*.

The initial implementation can use `rsync` to do the bulk-push operation. However
`rsync` is not the most efficient. The *host* is truth. Either through `slurp` or
the *editor*'s direct saving of changes, all edits
have reached the *host*. So: bulk-push can asume that 
the state of the *client* is exactly what was written by the bulk-push with the addition
of any changes that have since been `slurp`-ed. Asusme that the `slurp`-ed changes
are small. In this case: if bulk-push records what was previously pushed, it can
push changes based only on local (i.e. on the *host*) state. So, the ideal implementation
of the bulk-push would work like this:

* walk the *host* tree
* push each changed file not recorded in the bulk-push index
* an agent on the *client* writes the file updates provided by the push process

I can implement a faster bulk-push by reusing chunks of Kopia: 

* parallel walker
* index of what's shoved to the remote already (indexed by the hash)
* special "remote" that receives and writes the diffs
* if I code this right, it would even support files that mark what should be ignored.
